package hls

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bilalthdeveloper/kadrion/internal/core"
	"github.com/bilalthdeveloper/kadrion/utils"
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
	utils.WelComePrint(fmt.Sprintf("Addr Given %v", addr), fmt.Sprintf("Count Given %v", result.InitialCount), fmt.Sprintf("Duration Given %v", d), fmt.Sprintf("PumpCount %v", PumpCount))

	for {

		for i := 0; i <= int(result.InitialCount); i++ {
			go func() {
				var mu sync.Mutex
				mu.Lock()
				HlsIoLoop(ctx, addr, signal, d, counter)
				mu.Unlock()
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

		client := &gohlslib.Client{
			URI: addr,
		}
		err := client.Start()
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}
		waitCh := client.Wait()
		timeout := time.After(duration)
		for {
			select {
			case <-timeout:
				client.Close()
				signal <- 2
				counter.Add(1)
				return
			case err := <-waitCh:
				if err != nil {
					signal <- 1
					counter.Add(1)
					return
				}

			}
		}
	} else {

		utils.LogMessage("Timeout should be > 1 second for Hls", utils.Fatal_Error_Code)

	}
}
