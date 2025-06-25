package pool

import (
	"context"
	"fmt"
	"time"
)

type Pool struct {
	Workers        map[int]*Worker
	ResultCallback func(...interface{})
	ErrorCallback  func(error)
	TaskBuffer     *RingBuffer
	ctx            context.Context
	cancel         context.CancelFunc
	scaleTicker    *time.Ticker
}

func NewPool(workerCount int, resultCallback func(...interface{}), errorCallback func(error)) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		Workers:        make(map[int]*Worker),
		ResultCallback: resultCallback,
		ErrorCallback:  errorCallback,
		TaskBuffer:     NewRingBuffer(10000),
		ctx:            ctx,
		cancel:         cancel,
	}

	for i := 0; i < workerCount; i++ {
		worker := &Worker{
			Id:     i,
			Status: make(chan bool),
		}
		pool.Workers[i] = worker
	}

	return pool
}

func (p *Pool) AddTask(task Task) {
	p.TaskBuffer.Push(task)
}
func (p *Pool) Dispatch() {
	for _, worker := range p.Workers {
		go worker.Run(p)
	}
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-p.scaleTicker.C:
				p.adjustWorkerCount()
			}
		}
	}()

}
func (p *Pool) adjustWorkerCount() {

	bufferUsage := p.TaskBuffer.Usage()
	currentWorkers := len(p.Workers)

	if bufferUsage >= 0.8 {
		newWorker := &Worker{
			Id:     currentWorkers,
			Status: make(chan bool),
		}
		p.Workers[currentWorkers] = newWorker

		go newWorker.Run(p)

		LogMessage(fmt.Sprintf("Scaled up: %d now %s", currentWorkers+1, "workers"), 3)
	} else if bufferUsage < 0.25 {
		p.Workers[0].Status <- false
		LogMessage(fmt.Sprintf("Scaled Down: %d now %s", currentWorkers+1, "workers"), 3)
	}
}

func (p *Pool) Shutdown() {
	p.cancel()
}
