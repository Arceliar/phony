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
				select {
				case workerIn <- queue[0]:
					queue = queue[1:]
				case f := <-schedIn:
					queue = append(queue, f)
				}
			}
			queue = append(queue, <-schedIn)
		}
	}()
}
