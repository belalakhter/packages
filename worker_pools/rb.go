package pool

import "sync"

type RingBuffer struct {
	data       []func(...interface{}) (interface{}, error)
	size       int
	start, end int
	count      int
	lock       sync.Mutex
	notEmpty   *sync.Cond
	notFull    *sync.Cond
}

func NewRingBuffer(size int) *RingBuffer {
	rb := &RingBuffer{
		data: make([]func(...interface{}) (interface{}, error), size),
		size: size,
	}
	rb.notEmpty = sync.NewCond(&rb.lock)
	rb.notFull = sync.NewCond(&rb.lock)
	return rb
}

func (rb *RingBuffer) Push(task func(...interface{}) (interface{}, error)) {
	rb.lock.Lock()
	defer rb.lock.Unlock()
	for rb.count == rb.size {
		rb.notFull.Wait()
	}
	rb.data[rb.end] = task
	rb.end = (rb.end + 1) % rb.size
	rb.count++
	rb.notEmpty.Signal()
}

func (rb *RingBuffer) Pop() func(...interface{}) (interface{}, error) {
	rb.lock.Lock()
	defer rb.lock.Unlock()
	for rb.count == 0 {
		rb.notEmpty.Wait()
	}
	task := rb.data[rb.start]
	rb.start = (rb.start + 1) % rb.size
	rb.count--
	rb.notFull.Signal()
	return task
}

func (rb *RingBuffer) Usage() float64 {
	rb.lock.Lock()
	defer rb.lock.Unlock()
	return float64(rb.count) / float64(rb.size)
}
