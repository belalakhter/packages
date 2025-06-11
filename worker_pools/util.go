package pool

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

const (
	fatal_Error_Code = 1
	debug_Error_Code = 2
	log_Info         = 3
)

type Option func(*Pool)

func WithTimeout(timeout time.Duration) Option {
	return func(p *Pool) {
		p.timeout = timeout
	}
}

func WithAdjustInterval(interval time.Duration) Option {
	return func(p *Pool) {
		p.adjustInterval = interval
	}
}

func WithResultCallback(callback func(interface{})) Option {
	return func(p *Pool) {
		p.resultCallback = callback
	}
}

func WithErrorCallback(callback func(error)) Option {
	return func(p *Pool) {
		p.errorCallback = callback
	}
}

func WithRetryCount(retryCount int) Option {
	return func(p *Pool) {
		p.retryCount = retryCount
	}
}

func WithTaskQueueSize(size int) Option {
	return func(p *Pool) {
		p.taskQueueSize = size
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func logMessage(message string, code int) {
	switch code {
	case fatal_Error_Code:
		_, file, line, ok := runtime.Caller(1)
		if ok {
			fmt.Printf("\nFATAL ERROR: %s\nFile: %s, Line: %d\n", message, file, line)
		} else {
			fmt.Printf("\nFATAL ERROR: %s\n", message)
		}
		fmt.Println("The application will exit now.")
		os.Exit(1)
	case debug_Error_Code:
		fmt.Printf("\nDEBUG: %s\n", message)
	case log_Info:
		fmt.Printf("\nINFO: %s\n", message)
	default:
		fmt.Println("Unknown error code.")
	}
}
