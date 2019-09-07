package phony

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// pool of old messages, to avoid needing to allocate them every send
var pool = sync.Pool{New: func() interface{} { return new(queueElem) }}

// A message in the queue
type queueElem struct {
	msg  interface{}    // func() or func() bool
	next unsafe.Pointer // *queueElem, accessed atomically
}

// Inbox is an ordered queue of messages which an Actor will process sequentially.
// Messages are meant to be in the form of non-blocking functions of 0 arguments, often closures.
// The intent is for the Inbox struct to be embedded in other structs to satisfy the Actor interface, where the other fields of the struct are owned by the Actor.
// It is up to the user to ensure that memory is used safely, and that messages do not contain blocking operations.
// An Inbox must not be copied after first use.
type Inbox struct {
	head  *queueElem     // Used carefully to avoid needing atomics
	tail  unsafe.Pointer // *queueElem, accessed atomically
	count uint32         // updated atomically when a message is enqueued
}

// Actor is the interface for Actors, based on their ability to receive a message from another Actor.
// It's meant so that structs which embed an Inbox can satisfy a mutually compatible interface for message passing.
type Actor interface {
	Act(Actor, func())
	enqueue(interface{}) bool
	restart()
	advance() bool
}

// enqueue puts a message on the Actor's inbox queue and returns the number of messages that have been enqueued since the inbox was last empty.
// If the inbox was empty, then the actor was not already running, so enqueue starts it.
func (a *Inbox) enqueue(msg interface{}) bool {
	if msg == nil {
		panic("tried to send nil message")
	}
	q := pool.Get().(*queueElem)
	*q = queueElem{msg: msg}
	tail := (*queueElem)(atomic.SwapPointer(&a.tail, unsafe.Pointer(q)))
	if tail != nil {
		//An old tail exists, so update its next pointer to reference q
		atomic.StorePointer(&tail.next, unsafe.Pointer(q))
	} else {
		// No old tail existed, so no worker is currently running
		// Update the head to point to q, then start the worker
		a.head = q
		a.restart()
	}
	return atomic.AddUint32(&a.count, 1) > backpressureThreshold
}

// Act adds a message to an Actor's Inbox which tells the Actor to execute the provided function at some point in the future.
// When one Actor sends a message to another, the sender is meant to provide itself as the first argument to this function.
// If the receiver's Inbox has collected too many messages since it was last empty, and the sender argument is non-nil, then the sender is scheduled to pause at a safe point in the future until the receiver has finished running the action.
// A nil first argument is valid, and will prevent any scheduling changes from happening, in cases where an Actor wants to send a message to itself (where this scheduling is just useless overhead) or must receive a message from non-Actor code.
func (a *Inbox) Act(from Actor, action func()) {
	if a.enqueue(action) && from != nil {
		var s stop
		a.enqueue(func() {
			if !s.stop() && from.advance() {
				from.restart()
			}
		})
		from.enqueue(s.stop)
	}
}

// Block adds a message to an Actor's Inbox which tells the Actor to execute the provided function at some point in the future.
// It then blocks until the actor has finished running the provided function.
// Block meant exclusively as a convenience function for non-Actor code to send messages and wait for responses.
// If an Actor calls Block, then it may cause a deadlock, so Act should always be used instead.
func Block(actor Actor, action func()) {
	done := make(chan struct{})
	actor.Act(nil, func() { action(); close(done) })
	<-done
}

// run is executed when a message is placed in an empty Inbox, and launches a worker goroutine.
// The worker goroutine processes messages from the Inbox until empty, and then exits.
func (a *Inbox) run() {
	running := true
	for running {
		switch msg := a.head.msg.(type) {
		case func() bool: // used internally by backpressure
			if msg() {
				return
			}
		case func(): // all external use from Act
			msg()
		}
		running = a.advance()
	}
}

// returns true if we still have more work to do
func (a *Inbox) advance() bool {
	head := a.head
	atomic.AddUint32(&a.count, ^uint32(0)) // decrement counter
	for {
		a.head = (*queueElem)(atomic.LoadPointer(&head.next))
		if a.head != nil {
			// Move to the next message
			*head = queueElem{} // Clear fields before putting into pool
			pool.Put(head)
			return true // more left to do
		} else if !atomic.CompareAndSwapPointer(&a.tail, unsafe.Pointer(head), nil) {
			// The head is not the tail, but there was no head.next when we checked
			// Somebody must be updating it right now, so try again
			continue
		} else {
			// Head and tail are now both nil, our work here is done, exit
			*head = queueElem{} // Clear fields before putting into pool
			pool.Put(head)
			return false // done processing messages
		}
	}
}

func (a *Inbox) restart() {
	go a.run()
}

func (a *Inbox) cancel() {
	panic(isCancelled)
}

var isCancelled = new(struct{})

type stop uint32

func (s *stop) stop() bool {
	return atomic.SwapUint32((*uint32)(s), 1) == 0
}
