package pool

type Task func(...interface{}) (interface{}, error)
type Worker struct {
	Id     int
	Status chan bool
}

func (w *Worker) Run(pool *Pool) {
	w.Status <- true
	for {
		select {
		case <-pool.ctx.Done():
			return
		case status := <-w.Status:
			if !status {
				return
			}
		default:
			task := pool.TaskBuffer.Pop()
			if task != nil {
				res, err := w.ExecuteTask(task)
				w.HandleResult(res, err, pool)
			}
		}
	}
}

func (w *Worker) ExecuteTask(task Task) (result interface{}, err error) {
	return task()
}

func (w *Worker) HandleResult(result interface{}, err error, pool *Pool) {
	if err != nil && pool.ErrorCallback != nil {
		pool.ErrorCallback(err)
	} else if pool.ResultCallback != nil {
		pool.ResultCallback(result)
	}
}
