package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/belalakhter/work_utils/api_tester/internal/core"
	"github.com/belalakhter/work_utils/api_tester/utils"
)

func RunSseTest(ctx context.Context, addr string, initialCount int64, PumpCount int64, duration int64) {
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
		RunHttpTest(ctx, result, PumpCount, addr, Signal, duration, &global)
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

func RunHttpTest(ctx context.Context, result core.Result, PumpCount int64, addr string, signal chan int, d int64, counter *atomic.Uint64) {
	utils.WelComePrint(
		fmt.Sprintf("Addr Given %v", addr),
		fmt.Sprintf("Count Given %v", result.InitialCount),
		fmt.Sprintf("Duration Given %v", d),
		fmt.Sprintf("PumpCount %v", PumpCount),
	)

	for {
		for i := 0; i < int(result.InitialCount); i++ {
			go func() {
				SseIoLoop(ctx, addr, signal, d, counter)
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

func SseIoLoop(ctx context.Context, addr string, signal chan int, d int64, counter *atomic.Uint64) {
	if d > 1 {
		duration := time.Second * time.Duration(d)

		connCtx, cancel := context.WithTimeout(ctx, duration+time.Second*2)
		defer cancel()

		req, err := http.NewRequestWithContext(connCtx, "GET", addr, nil)
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}

		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")

		client := &http.Client{
			Timeout: duration + time.Second*2,
		}

		resp, err := client.Do(req)
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			signal <- 1
			counter.Add(1)
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		timeout := time.After(duration)

		dataReceived := false
		lastDataTime := time.Now()

		for {
			select {
			case <-timeout:

				signal <- 2
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
			default:

				select {
				case <-time.After(time.Millisecond * 500):
					if time.Since(lastDataTime) > time.Second*5 {
						signal <- 1
						counter.Add(1)
						return
					}
					continue
				default:
					if scanner.Scan() {

						dataReceived = true
						lastDataTime = time.Now()

						line := scanner.Text()

						if line == "" || line[:5] == "data:" || line[:6] == "event:" || line[:3] == "id:" {

							continue
						}
					}

					if err := scanner.Err(); err != nil {
						signal <- 1
						counter.Add(1)
						return
					}

					if !scanner.Scan() && scanner.Err() == nil {
						if dataReceived {
							signal <- 2
						} else {
							signal <- 1
						}
						counter.Add(1)
						return
					}
				}
			}
		}
	} else {
		utils.LogMessage("Timeout should be > 1 second for SSE", utils.Fatal_Error_Code)
		signal <- 1
		counter.Add(1)
	}
}
