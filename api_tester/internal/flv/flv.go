package flv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/belalakhter/work_utils/tree/main/api_tester/internal/core"
	"github.com/belalakhter/work_utils/tree/main/api_tester/utils"
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
	utils.WelComePrint(
		fmt.Sprintf("Addr Given %v", addr),
		fmt.Sprintf("Count Given %v", result.InitialCount),
		fmt.Sprintf("Duration Given %v", d),
		fmt.Sprintf("PumpCount %v", PumpCount),
	)

	for {
		for i := 0; i < int(result.InitialCount); i++ {
			go func(index int) {
				FlvIoLoop(ctx, addr, signal, d, counter, index)
			}(i)
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

func FlvIoLoop(ctx context.Context, addr string, signal chan int, d int64, counter *atomic.Uint64, index int) {
	if d <= 0 {
		utils.LogMessage("Timeout should be > 0 seconds for FLV", utils.Fatal_Error_Code)
		signal <- 1
		counter.Add(1)
		return
	}

	duration := time.Second * time.Duration(d)

	filename := fmt.Sprintf("output_%d_%d.flv", time.Now().UnixNano(), index)
	file, err := os.Create(filename)
	if err != nil {
		utils.LogMessage(fmt.Sprintf("Failed to create file %s: %v", filename, err), utils.Log_Info)
		signal <- 1
		counter.Add(1)
		return
	}
	defer func() {
		file.Close()
		os.Remove(filename)
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, duration+time.Second*5)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", addr, nil)
	if err != nil {
		utils.LogMessage(fmt.Sprintf("Failed to create request: %v", err), utils.Log_Info)
		signal <- 1
		counter.Add(1)
		return
	}

	client := &http.Client{
		Timeout: duration + time.Second*5,
	}

	resp, err := client.Do(req)
	if err != nil {
		utils.LogMessage(fmt.Sprintf("Failed to connect to %s: %v", addr, err), utils.Log_Info)
		signal <- 1
		counter.Add(1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.LogMessage(fmt.Sprintf("HTTP error: %d %s", resp.StatusCode, resp.Status), utils.Log_Info)
		signal <- 1
		counter.Add(1)
		return
	}

	writer := CustomWriter(addr, file)
	defer writer.Close()

	go writer.processPackets()

	buffer := make([]byte, 8192)
	timeout := time.After(duration)
	bytesReceived := int64(0)
	packetsWritten := int64(0)
	lastActivity := time.Now()

	healthTicker := time.NewTicker(time.Second * 2)
	defer healthTicker.Stop()

	for {
		select {
		case <-timeout:
			if bytesReceived > 0 {
				utils.LogMessage(fmt.Sprintf("FLV client completed successfully. Bytes: %d, Packets: %d", bytesReceived, packetsWritten), utils.Log_Info)
				signal <- 2
			} else {
				utils.LogMessage("FLV client completed but no data received", utils.Log_Info)
				signal <- 1
			}
			counter.Add(1)
			return

		case <-timeoutCtx.Done():
			if bytesReceived > 0 {
				signal <- 2
			} else {
				signal <- 1
			}
			counter.Add(1)
			return

		case <-healthTicker.C:
			if bytesReceived == 0 && time.Since(lastActivity) > time.Second*8 {
				utils.LogMessage("FLV health check failed: no data received", utils.Log_Info)
				signal <- 1
				counter.Add(1)
				return
			}

		default:

			if conn, ok := resp.Body.(interface{ SetReadDeadline(time.Time) error }); ok {
				conn.SetReadDeadline(time.Now().Add(time.Second * 5))
			}

			n, err := resp.Body.Read(buffer)
			if err != nil {
				if err == io.EOF {
					if bytesReceived > 0 {
						utils.LogMessage(fmt.Sprintf("FLV stream ended. Bytes: %d, Packets: %d", bytesReceived, packetsWritten), utils.Log_Info)
						signal <- 2
					} else {
						utils.LogMessage("FLV stream ended but no data received", utils.Log_Info)
						signal <- 1
					}
					counter.Add(1)
					return
				}
				utils.LogMessage(fmt.Sprintf("FLV read error: %v", err), utils.Log_Info)
				signal <- 1
				counter.Add(1)
				return
			}

			if n > 0 {
				bytesReceived += int64(n)
				lastActivity = time.Now()

				packet := &av.Packet{
					Data:      make([]byte, n),
					TimeStamp: uint32(time.Now().UnixMilli()),
					IsVideo:   true,
				}
				copy(packet.Data, buffer[:n])

				if err := writer.Write(packet); err != nil {
					utils.LogMessage(fmt.Sprintf("Failed to write packet: %v", err), utils.Log_Info)
				} else {
					packetsWritten++
				}
			}

			time.Sleep(time.Millisecond * 10)
		}
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

	flvHeader := []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09}
	if _, err := ret.writer.Write(flvHeader); err != nil {
		utils.LogMessage(fmt.Sprintf("Error writing FLV header: %v", err), utils.Log_Info)
		ret.closed = true
		return ret
	}

	pio.PutI32BE(ret.buf[:4], 0)
	if _, err := ret.writer.Write(ret.buf[:4]); err != nil {
		utils.LogMessage(fmt.Sprintf("Error writing FLV previous tag size: %v", err), utils.Log_Info)
		ret.closed = true
		return ret
	}

	return ret
}

func (flvWriter *FLVWriter) Write(p *av.Packet) error {
	if flvWriter.closed {
		return fmt.Errorf("FLVWriter is closed")
	}

	select {
	case flvWriter.packetQueue <- p:
		return nil
	case <-time.After(time.Millisecond * 100):
		return fmt.Errorf("packet queue timeout")
	}
}

func (flvWriter *FLVWriter) processPackets() {
	defer func() {
		if r := recover(); r != nil {
			utils.LogMessage(fmt.Sprintf("Recovered from panic in processPackets: %v", r), utils.Log_Info)
		}
	}()

	for {
		select {
		case packet, ok := <-flvWriter.packetQueue:
			if !ok {
				return
			}

			if packet != nil && len(packet.Data) > 0 {

				if _, err := flvWriter.writer.Write(packet.Data); err != nil {
					utils.LogMessage(fmt.Sprintf("Error writing packet data: %v", err), utils.Log_Info)
					return
				}
			}

		case <-flvWriter.closedChan:
			return
		}
	}
}

func (flvWriter *FLVWriter) Close() {
	if !flvWriter.closed {
		flvWriter.closed = true
		close(flvWriter.closedChan)
		close(flvWriter.packetQueue)
	}
}
