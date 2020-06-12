package phony

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// pool of old queueElems, to avoid needing to allocate them every send
// considering how small a queueElem is, this may not be worth it
var pool = sync.Pool{New: func() interface{} { return new(queueElem) }}

// A message in the queue
type queueElem struct {
	msg  func()
	next unsafe.Pointer // *queueElem, accessed atomically
}

// Inbox is an ordered queue of messages which an Actor will process sequentially.
// Messages are meant to be in the form of non-blocking functions of 0 arguments, often closures.
// The intent is for the Inbox struct to be embedded in other structs, causing them to satisfy the Actor interface, and then the Actor is used to access any protected fields of the struct.
// It is up to the user to ensure that memory is used safely, and that messages do not contain blocking operations.
// An Inbox must not be copied after first use.
type Inbox struct {
	head *queueElem     // Used carefully to avoid needing atomics
	tail unsafe.Pointer // *queueElem, accessed atomically
	busy uint32         // accessed atomically, 1 if sends should apply backpressure
}

// Actor is the interface for Actors, based on their ability to receive a message from another Actor.
// It's meant so that structs which embed an Inbox can satisfy a mutually compatible interface for message passing.
type Actor interface {
	Act(Actor, func())
	enqueue(func())
	restart()
	advance() bool
}

// enqueue puts a message into the Inbox and returns true if backpressure should be applied.
// If the inbox was empty, then the actor was not already running, so enqueue starts it.
func (a *Inbox) enqueue(msg func()) {
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
}

// Act adds a message to an Inbox, which will be executed by the inbox's Actor at some point in the future.
// When one Actor sends a message to another, the sender is meant to provide itself as the first argument to this function.
// If the sender argument is non-nil and the receiving Inbox has been flooded, then backpressure is applied to the sender.
// This backpressue cause the sender stop processing messages at some point in the future until the receiver has caught up with the sent message.
// A nil first argument is valid, but should only be used in cases where backpressure is known to be unnecessary, such as when an Actor sends a message to itself or when non-Actor code sends a message to an Actor (where it's used internally by Block).
func (a *Inbox) Act(from Actor, action func()) {
	if action == nil {
		panic("tried to send nil action")
	}
	a.enqueue(action)
	if from != nil && atomic.LoadUint32(&a.busy) != 0 {
		s := stop{from: from}
		a.enqueue(s.signal)
		from.enqueue(s.wait)
	}
}

// Block adds a message to an Actor's Inbox, which will be executed at some point in the future.
// It then blocks until the Actor has finished running the provided function.
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
	atomic.StoreUint32(&a.busy, 1)
	for running := true; running; running = a.advance() {
		a.head.msg()
	}
}

// returns true if we still have more work to do
func (a *Inbox) advance() (more bool) {
	head := a.head
	a.head = (*queueElem)(atomic.LoadPointer(&head.next))
	if a.head == nil {
		// We loaded the last message
		// Unset busy and CAS the tail to nil to shut down
		atomic.StoreUint32(&a.busy, 0)
		if !atomic.CompareAndSwapPointer(&a.tail, unsafe.Pointer(head), nil) {
			// Someone pushed to the list before we could CAS the tail to shut down
			// This means we're effectively restarting at this point
			// Set busy and load the next message
			atomic.StoreUint32(&a.busy, 1)
			for a.head == nil {
				// Busy loop until the message is successfully loaded
				// Gosched to avoid blocking the thread in the mean time
				runtime.Gosched()
				a.head = (*queueElem)(atomic.LoadPointer(&head.next))
			}
			more = true
		}
	} else {
		more = true
	}
	*head = queueElem{} // clear fields
	pool.Put(head)
	return
}

func (a *Inbox) restart() {
	go a.run()
}

type stop struct {
	flag uint32
	from Actor
}

func (s *stop) signal() {
	if atomic.SwapUint32((*uint32)(&s.flag), 1) != 0 && s.from.advance() {
		s.from.restart()
	}
}

func (s *stop) wait() {
	if atomic.SwapUint32((*uint32)(&s.flag), 1) == 0 {
		runtime.Goexit()
	}
}
