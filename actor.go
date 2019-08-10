package gony

import (
	"sync"
)

type Actor struct {
	mutex  sync.Mutex
	acting bool
	queue  []func()
}

func (a *Actor) SendMessageTo(destination *Actor, message func()) {
	// Ideally, we would compare lengths atomically, somehow
	ch := make(chan struct{})
	f := func() { message(); close(ch) }
	destination.mutex.Lock()
	dLen := len(destination.queue)
	destination.queue = append(destination.queue, f)
	if !destination.acting {
		destination.acting = true
		go destination.doActions()
	}
	destination.mutex.Unlock()
	a.mutex.Lock()
	if len(a.queue) < 4*dLen {
		a.queue = append(a.queue, func() { <-ch })
	}
	a.mutex.Unlock()
}

func (a *Actor) Act(f func()) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.queue = append(a.queue, f)
	if !a.acting {
		a.acting = true
		go a.doActions()
	}
}

func (a *Actor) doActions() {
	for {
		var f func()
		a.mutex.Lock()
		if len(a.queue) > 0 {
			f, a.queue = a.queue[0], a.queue[1:]
		} else {
			a.acting = false
		}
		a.mutex.Unlock()
		if f != nil {
			f()
		} else {
			return
		}
	}
}
