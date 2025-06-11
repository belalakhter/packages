package flv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bilalthdeveloper/kadrion/internal/core"
	"github.com/bilalthdeveloper/kadrion/utils"
	"github.com/gwuhaolin/livego/av"
	"github.com/gwuhaolin/livego/utils/pio"
)

const (
	headerLen   = 11
	maxQueueNum = 1024
)

type FLVWriter struct {
	av.RWBaser
	buf         []byte
	closed      bool
	closedChan  chan struct{}
	writer      io.Writer
	packetQueue chan *av.Packet
}

func RunFlvTest(ctx context.Context, addr string, initialCount int64, PumpCount int64, duration int64) {
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
				FlvIoLoop(ctx, addr, signal, d, counter)
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

func FlvIoLoop(ctx context.Context, addr string, signal chan int, d int64, counter *atomic.Uint64) {

	if d > 0 {

		duration := time.Second * time.Duration(d)
		file, err := os.Create("output.flv")
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}
		defer file.Close()
		writer := CustomWriter(addr, file)
		resp, err := http.Get(addr)
		if err != nil {
			signal <- 1
			counter.Add(1)
			return
		}
		defer resp.Body.Close()
		buffer := make([]byte, 8192)
		timeout := time.After(duration)
		for {
			select {
			case <-timeout:
				writer.Close()
				signal <- 2
				counter.Add(1)
				return
			default:
				n, err := resp.Body.Read(buffer)
				if err != nil {
					if err == io.EOF {
						signal <- 2
						counter.Add(1)
						return
					}
					signal <- 1
					counter.Add(1)
					return
				}

				packet := &av.Packet{
					Data:      buffer[:n],
					TimeStamp: uint32(time.Now().Unix()),
					IsVideo:   true,
				}

				if err := writer.Write(packet); err != nil {

				}
				utils.LogMessage(fmt.Sprintf("Wrote Stream to file", len(buffer)), 3)
			}
		}
	} else {

		utils.LogMessage("Timeout should be > 1 second for Hls", utils.Fatal_Error_Code)

	}
}
func CustomWriter(url string, writer io.Writer) *FLVWriter {
	ret := &FLVWriter{
		writer:      writer,
		RWBaser:     av.NewRWBaser(time.Second * 10),
		closedChan:  make(chan struct{}),
		buf:         make([]byte, headerLen),
		packetQueue: make(chan *av.Packet, maxQueueNum),
	}

	if _, err := ret.writer.Write([]byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09}); err != nil {
		fmt.Printf("Error writing FLV header: %v", err)
		ret.closed = true
	}
	pio.PutI32BE(ret.buf[:4], 0)
	if _, err := ret.writer.Write(ret.buf[:4]); err != nil {
		fmt.Printf("Error writing FLV header: %v", err)
		ret.closed = true
	}

	return ret
}
func (flvWriter *FLVWriter) Write(p *av.Packet) error {
	if flvWriter.closed {
		return fmt.Errorf("FLVWriter is closed")
	}

	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("Recovered from panic in Write: %v", e)
			flvWriter.closed = true
		}
	}()

	if len(flvWriter.packetQueue) >= maxQueueNum-24 {
		return fmt.Errorf("packet queue is full")
	}

	select {
	case flvWriter.packetQueue <- p:
		return nil
	default:
		return fmt.Errorf("failed to enqueue packet")
	}
}

func (flvWriter *FLVWriter) Close() {
	if !flvWriter.closed {
		close(flvWriter.packetQueue)
		close(flvWriter.closedChan)
		flvWriter.closed = true
	}
}
