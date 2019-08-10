package gony

import "sync"

/* TODO backpressure:

Compare queue sizes when an actor sends a message to another actor.
If the sender's queue is less than 4 times the receiver's queue, then we say
that the receiver is overworked and apply backpressure.

To apply backpressure, we create a channel and send an additional message to the receiver.
All this message does is close the channel.
*After* that (causal messaging!) we send a message to ourself that blocks reading from the channel.
This appears to be safe (deadlock free) and gives the receiver time to finish some work before we keep going / lets our own queue build up a bit to even things out.

Then normal go code acting from the outside would need to use similar tricks (maybe we provide helpers to make it easier) to block until a message is handled.

*/

type Actor struct {
	// TODO use linked lists bult from CAS instead of a mutex and slice
	mutex  sync.Mutex
	acting bool
	queue  []func()
}

func (a *Actor) SendTo(destination *Actor, message func()) {
	// Ideally, we would compare lengths atomically
	//  I don't know of a deadlock-free way to do that...
	//  Or at least, not with the current queue implementation...
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
