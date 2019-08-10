package gony

import (
	"sync"
)

type Actor struct {
	mutex   sync.Mutex
	running bool
	queue   []func()
}

// Adds an item to the queue and returns the new queue size
func (a *Actor) enqueue(f func()) int {
	if f == nil {
		panic("tried to send nil message")
	}
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.queue = append(a.queue, f)
	if !a.running {
		a.running = true
		go a.run()
	}
	return len(a.queue)
}

func (a *Actor) SyncExec(f func()) {
	done := make(chan struct{})
	a.enqueue(func() { f(); close(done) })
	<-done
}

func (a *Actor) SendMessageTo(destination *Actor, message func()) {
	// Ideally, we would compare lengths atomically, somehow
	dLen := destination.enqueue(message)
	a.mutex.Lock()
	aLen := len(a.queue)
	a.mutex.Unlock()
	if 4*aLen < dLen {
		// Tried to send to a much larger queue, so add some backpressure
		done := make(chan struct{})
		destination.enqueue(func() { close(done) })
		a.enqueue(func() { <-done })
	}
}

func (a *Actor) run() {
	for {
		var f func()
		a.mutex.Lock()
		if len(a.queue) > 0 {
			f, a.queue = a.queue[0], a.queue[1:]
		} else {
			a.running = false
		}
		a.mutex.Unlock()
		if f != nil {
			f()
		} else {
			return
		}
	}
}
