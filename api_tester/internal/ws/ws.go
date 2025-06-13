package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
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
	utils.WelComePrint(
		fmt.Sprintf("Addr Given %v", addr),
		fmt.Sprintf("Count Given %v", result.InitialCount),
		fmt.Sprintf("Duration Given %v", d),
		fmt.Sprintf("PumpCount %v", PumpCount),
	)

	for {
		for i := 0; i < int(result.InitialCount); i++ {
			go func() {
				WsIoLoop(ctx, addr, signal, d, counter)
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

		connCtx, cancel := context.WithTimeout(ctx, duration+time.Second*2) // Add buffer for connection setup
		defer cancel()

		conn, _, _, err := ws.Dial(connCtx, addr)
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}
		defer conn.Close()

		timeout := time.After(duration)

		for {
			select {
			case <-timeout:
				signal <- 2
				counter.Add(1)
				return
			case <-connCtx.Done():
				signal <- 1
				counter.Add(1)
				return
			default:
				conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
				_, _, err := wsutil.ReadServerData(conn)
				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						continue
					}
					signal <- 1
					counter.Add(1)
					return
				}
			}
		}
	} else {
		conn, _, _, err := ws.Dial(ctx, addr)
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}
		defer conn.Close()

		err = wsutil.WriteClientMessage(conn, ws.OpText, []byte("Ping"))
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}

		time.Sleep(time.Millisecond * 100)
		signal <- 2
		counter.Add(1)
	}
}
