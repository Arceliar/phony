package gony

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

/* TODO backpressure:

Compare queue sizes when an actor sends a message to another actor.
If the sender's queue is less than 4 times the receiver's queue, then we say
that the receiver is overworked and apply backpressure.

To apply backpressure, we create a channel and send an additional message to the receiver.
All this message does is close the channel.
*After* that (causal messaging!) we send a message to ourself that blocks reading from the channel.
This appears to be safe (deadlock free) and gives the receiver time to finish some work before we keep going / lets our own queue build up a bit to even things out.

Then normal go code acting from the outside would need to use similar tricks (maybe we provide helpers to make it easier) to block until a message is handled.

OK, that's more-or-less implemented now, so the next step is to use atomic CAS operations
(since those should be faster than mutexes at least)

*/

type queueElem struct {
	message func()
	_next   unsafe.Pointer // *queueElem accessed atomically
}

func (q *queueElem) getNext() *queueElem {
	var next *queueElem
	if p := atomic.LoadPointer(&q._next); p != nil {
		next = (*queueElem)(p)
	}
	return next
}

func (q *queueElem) putNext(oldNext *queueElem, newNext *queueElem) bool {
	oldP := unsafe.Pointer(oldNext)
	newP := unsafe.Pointer(newNext)
	return atomic.CompareAndSwapPointer(&q._next, oldP, newP)
}

type queue struct {
	head *queueElem
	tail *queueElem
	size int
}

type Actor struct {
	_inbox unsafe.Pointer // *queue accessed atomically
}

func (a *Actor) getInbox() *queue {
	var q *queue
	if p := atomic.LoadPointer(&a._inbox); p != nil {
		q = (*queue)(p)
	}
	return q
}

func (a *Actor) putInbox(oldInbox *queue, newInbox *queue) bool {
	oldP := unsafe.Pointer(oldInbox)
	newP := unsafe.Pointer(newInbox)
	return atomic.CompareAndSwapPointer(&a._inbox, oldP, newP)
}

func (a *Actor) enqueue(msg func()) {
	tail := queueElem{message: msg}
	for {
		newInbox := queue{tail: &tail}
		oldInbox := a.getInbox()
		if oldInbox == nil {
			// This is the first message
			newInbox.head = &queueElem{_next: unsafe.Pointer(&tail)} // Dummy head
			newInbox.size++
			if !a.putInbox(oldInbox, &newInbox) {
				continue
			}
			// The actor wasn't running, so start it and return.
			go a.run()
			return
		}
		// Try to add ourself to the queue
		if !oldInbox.tail.putNext(nil, &tail) {
			// Something updated the queue in the mean time, start over
			continue
		}
		// We added ourself to the tail, so now we need to update the inbox accordingly
		for {
			newInbox.head = oldInbox.head
			newInbox.size = oldInbox.size + 1
			if !a.putInbox(oldInbox, &newInbox) {
				// The worker advanced the inbox in the mean time, so try again
				// Note that nobody else could have advanced, because oldInbox.tail._next is non-nil
				oldInbox = a.getInbox()
				continue
			}
			// We've updated the inbox, we're done now
			break
		}
		if oldInbox.head == oldInbox.tail {
			// The worker exited, so restart it
			go a.run()
		}
		return
	}
}

// This worker runs in a goroutine and processes messages from the queue
func (a *Actor) run() {
	for {
		// Get the queue
		oldInbox := a.getInbox()
		// The head has already been run, so get the new head (old head's _next)
		head := oldInbox.head.getNext()
		// Handle the message
		head.message()
		// Clear the message so it can be GCed immediately, prepare new inbox
		head.message = nil
		// Now advance the queue in a CAS loop
		newInbox := queue{head: head}
		for {
			newInbox.tail = oldInbox.tail
			newInbox.size = oldInbox.size - 1
			if !a.putInbox(oldInbox, &newInbox) {
				// The old inbox changed in the mean time, try again
				oldInbox = a.getInbox()
				continue
			}
			break
		}
		if newInbox.head != newInbox.tail {
			// There are still messages to process, so keep going
			continue
		}
		break
	}
}

func (a *Actor) Act(msg func()) {
	a.enqueue(msg)
}

func (a *Actor) SendMessageTo(destination *Actor, message func()) {
	destination.enqueue(message)
	var mySize, theirSize int
	if q := a.getInbox(); q != nil {
		mySize = q.size
	}
	if q := destination.getInbox(); q != nil {
		theirSize = q.size
	}
	if mySize < 4*theirSize {
		// Backpressure
		done := make(chan struct{})
		destination.enqueue(func() { close(done) })
		a.enqueue(func() { <-done })
	}
}

////////////////////////////////////////////////////////////////////////////////

type Actor_old struct {
	// TODO use linked lists bult from CAS instead of a mutex and slice
	mutex  sync.Mutex
	acting bool
	queue  []func()
}

func (a *Actor_old) SendTo(destination *Actor_old, message func()) {
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

func (a *Actor_old) Act(f func()) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.queue = append(a.queue, f)
	if !a.acting {
		a.acting = true
		go a.doActions()
	}
}

func (a *Actor_old) doActions() {
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
