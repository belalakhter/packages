package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bilalthdeveloper/kadrion/internal/core"
	"github.com/bilalthdeveloper/kadrion/utils"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func RunWebsocketTest(ctx context.Context, addr string, initialCount int64, PumpCount int64, duration int64) {
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
		RunSocketTest(ctx, result, PumpCount, addr, Signal, duration, &global)
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

func RunSocketTest(ctx context.Context, result core.Result, PumpCount int64, addr string, signal chan int, d int64, counter *atomic.Uint64) {
	utils.WelComePrint(fmt.Sprintf("Addr Given %v", addr), fmt.Sprintf("Count Given %v", result.InitialCount), fmt.Sprintf("Duration Given %v", d), fmt.Sprintf("PumpCount %v", PumpCount))
	ctx, _ = context.WithTimeout(ctx, time.Second*10)

	for {
		for i := 0; i <= int(result.InitialCount); i++ {
			go func() {
				var mu sync.Mutex
				mu.Lock()
				WsIoLoop(ctx, addr, signal, d, counter)
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

func WsIoLoop(ctx context.Context, addr string, signal chan int, d int64, counter *atomic.Uint64) {
	if d > 0 {
		duration := time.Second * time.Duration(d)
		var conn net.Conn
		var err error
		conn, _, _, err = ws.Dial(ctx, addr)
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}

		timeout := time.After(duration)

		for {
			select {

			case <-timeout:

				err := conn.Close()
				if err != nil {
				}
				signal <- 2
				counter.Add(1)
				return

			default:
				_, _, err := wsutil.ReadServerData(conn)
				if err != nil {
					signal <- 1
					counter.Add(1)
					return
				}

			}

		}
	} else {
		var conn net.Conn
		var err error
		conn, _, _, err = ws.Dial(ctx, addr)
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}
		for {
			err = wsutil.WriteClientMessage(conn, ws.OpText, []byte("Ping"))
			if err != nil {
				signal <- 1
				counter.Add(1)
				return
			}
			time.Sleep(time.Millisecond * 100)
			err = conn.Close()
			if err != nil {
			}
			signal <- 2
			counter.Add(1)
			return
		}

	}
}
