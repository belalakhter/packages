## Features
- Minimal structure
- Scalability included
- Result and error handling
- Ring buffer for task management

## Usage
- go get github.com/belalakhter/packages/worker_pools@v1.0.6
- import "github.com/belalakhter/packages/worker_pools"
- use ```pool.New(10,resultHandler,errorHandler)``` and ```pool.Dispatch()```
- use pool.AddTask(func(... interface{}) (interface{},error)) to append tasks to the pool
