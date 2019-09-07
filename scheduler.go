package phony

import "runtime"

var schedIn chan func()

func schedule(f func()) {
	schedIn <- f
}

func init() {
	count := runtime.GOMAXPROCS(0)
	if count < 1 {
		count = 1
	}
	workerIn := make(chan func(), 2*count)
	schedIn = make(chan func(), 2*count)
	for idx := 0; idx < count; idx++ {
		go func() {
			for f := range workerIn {
				f()
			}
		}()
	}
	go func() {
		var queue []func()
		for {
			for len(queue) > 0 {
				idx := len(queue) - 1 // LIFO to avoid reallocating the queue slice too much
				select {
				case workerIn <- queue[idx]:
					queue = queue[:idx]
				case f := <-schedIn:
					queue = append(queue, f)
				}
			}
			queue = append(queue, <-schedIn)
		}
	}()
}
