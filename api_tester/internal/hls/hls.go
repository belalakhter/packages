package hls

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/belalakhter/work_utils/api_tester/internal/core"
	"github.com/belalakhter/work_utils/api_tester/utils"
	"github.com/bluenviron/gohlslib"
)

func RunHlsTest(ctx context.Context, addr string, initialCount int64, PumpCount int64, duration int64) {
	var global atomic.Uint64
	global.Store(0)
	Signal := make(chan int, 10000)

	result := core.Result{
		InitialCount: initialCount,
		Passed:       0,
		Failed:       0,
		StopCount:    utils.CalculateStopCount(initialCount, PumpCount),
	}

	go func() {
		RunBufferStreamTest(ctx, result, PumpCount, addr, Signal, duration, &global)
	}()

	for {
		sig := <-Signal
		switch sig {
		case 1:
			result.Failed++
		case 2:
			result.Passed++
		}

		if global.Load() == uint64(result.StopCount) {
			resp, err := json.Marshal(result)
			if err != nil {
				utils.LogMessage(err.Error(), utils.Fatal_Error_Code)
			}
			utils.LogMessage(string(resp), utils.Log_Info)
			break
		}
	}
}

func RunBufferStreamTest(ctx context.Context, result core.Result, PumpCount int64, addr string, signal chan int, d int64, counter *atomic.Uint64) {
	utils.WelComePrint(
		fmt.Sprintf("Addr Given %v", addr),
		fmt.Sprintf("Count Given %v", result.InitialCount),
		fmt.Sprintf("Duration Given %v", d),
		fmt.Sprintf("PumpCount %v", PumpCount),
	)

	for {
		for i := 0; i < int(result.InitialCount); i++ {
			go func() {
				HlsIoLoop(ctx, addr, signal, d, counter)
			}()
		}

		utils.LogMessage(fmt.Sprintf("Users Dispatched %v", result.InitialCount), 3)

		if PumpCount == 0 {
			break
		}

		PumpCount--
		result.InitialCount = result.InitialCount * 2
		time.Sleep(time.Second * 1)
	}
}

func HlsIoLoop(ctx context.Context, addr string, signal chan int, d int64, counter *atomic.Uint64) {
	if d > 0 {
		duration := time.Second * time.Duration(d)

		connCtx, cancel := context.WithTimeout(ctx, duration+time.Second*5) // Extra buffer for HLS startup
		defer cancel()

		client := &gohlslib.Client{
			URI: addr,
		}

		dataReceived := false
		segmentsReceived := 0

		client.OnTracks = func(tracks []*gohlslib.Track) error {
			utils.LogMessage(fmt.Sprintf("HLS tracks received: %d", len(tracks)), utils.Log_Info)

			for _, track := range tracks {
				client.OnDataH26x(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
					dataReceived = true
					segmentsReceived++
				})

				client.OnDataMPEG4Audio(track, func(pts time.Duration, aus [][]byte) {
					dataReceived = true
					segmentsReceived++
				})

				client.OnDataOpus(track, func(pts time.Duration, packets [][]byte) {
					dataReceived = true
					segmentsReceived++
				})

				client.OnDataVP9(track, func(pts time.Duration, frame []byte) {
					dataReceived = true
					segmentsReceived++
				})

				client.OnDataAV1(track, func(pts time.Duration, tu [][]byte) {
					dataReceived = true
					segmentsReceived++
				})
			}
			return nil
		}

		err := client.Start()
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}

		defer func() {
			client.Close()
		}()

		waitCh := client.Wait()
		timeout := time.After(duration)

		healthTicker := time.NewTicker(time.Second * 2)
		defer healthTicker.Stop()

		lastSegmentTime := time.Now()

		for {
			select {
			case <-timeout:
				if dataReceived {
					utils.LogMessage(fmt.Sprintf("HLS client completed successfully. Segments received: %d", segmentsReceived), utils.Log_Info)
					signal <- 2
				} else {
					utils.LogMessage("HLS client completed but no data received", utils.Log_Info)
					signal <- 1
				}
				counter.Add(1)
				return

			case <-connCtx.Done():
				if dataReceived {
					signal <- 2
				} else {
					signal <- 1
				}
				counter.Add(1)
				return

			case err := <-waitCh:
				if err != nil {
					utils.LogMessage(fmt.Sprintf("HLS client error: %v", err), utils.Log_Info)
					signal <- 1
				} else {
					if dataReceived {
						signal <- 2
					} else {
						signal <- 1
					}
				}
				counter.Add(1)
				return

			case <-healthTicker.C:
				if dataReceived {
					currentSegments := segmentsReceived
					if currentSegments > 0 {
						lastSegmentTime = time.Now()
					} else if time.Since(lastSegmentTime) > time.Second*10 {
						utils.LogMessage("HLS health check failed: no segments received for 10 seconds", utils.Log_Info)
						signal <- 1
						counter.Add(1)
						return
					}
				} else if time.Since(lastSegmentTime) > time.Second*8 {
					utils.LogMessage("HLS health check failed: no initial data received", utils.Log_Info)
					signal <- 1
					counter.Add(1)
					return
				}
			}
		}
	} else {
		utils.LogMessage("Timeout should be > 0 seconds for HLS", utils.Fatal_Error_Code)
		signal <- 1
		counter.Add(1)
	}
}
