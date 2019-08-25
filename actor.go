// Package phony is a small actor model library for Go, inspired by the causal messaging system in the Pony programming language.
// An Actor is an interface satisfied by a lightweight Inbox struct.
// Structs that embed an Inbox satisfy an interface that allows them to send messages to eachother.
// Messages are functions of 0 arguments, typically closures, and should not perform blocking operations.
// Message passing is asynchronous, causal, and fast.
// Actors implemented by the provided Inbox struct are scheduled to prevent messages queues from growing too large, by pausing at safe breakpoints when an Actor detects that it sent something to another Actor whose inbox is flooded.
package phony

import (
	"sync/atomic"
	"unsafe"
)

// How large a queue can be before backpressure slows down sending to it.
const backpressureThreshold = 127

// A message in the queue
type queueElem struct {
	msg   func()
	next  unsafe.Pointer // *queueElem, accessed atomically
	count uint
}

// Inbox is an ordered queue of messages which an Actor will process sequentially.
// Messages are meant to be in the form of non-blocking functions of 0 arguments, often closures.
// The intent is for the Inbox struct to be embedded in other structs to satisfy the Actor interface, where the other fields of the struct are owned by the Actor.
// It is up to the user to ensure that memory is used safely, and that messages do not contain blocking operations.
// An Inbox must not be copied after first use.
type Inbox struct {
	head *queueElem     // Used carefully to avoid needing atomics
	tail unsafe.Pointer // *queueElem, accessed atomically
}

// Actor is the interface for Actors, based on their ability to receive a message from another Actor.
// It's meant so that structs which embed an Inbox can satisfy a mutually compatible interface for message passing.
type Actor interface {
	RecvFrom(Actor, func())
}

// enqueue puts a message on the Actor's inbox queue and returns the number of messages that have been enqueued since the inbox was last empty.
// If the inbox was empty, then the actor was not already running, so enqueue starts it.
func (a *Inbox) enqueue(f func()) uint {
	if f == nil {
		panic("tried to send nil message")
	}
	q := &queueElem{msg: f}
	tail := (*queueElem)(atomic.SwapPointer(&a.tail, unsafe.Pointer(q)))
	if tail != nil {
		//An old tail exists, so update its next pointer to reference q
		q.count = tail.count + 1
		atomic.StorePointer(&tail.next, unsafe.Pointer(q))
	} else {
		// No old tail existed, so no worker is currently running
		// Update the head to point to q, then start the worker
		a.head = q
		go a.run()
	}
	return q.count
}

// RecvFrom adds a message to the Inbox and, if the Inbox is flooded, signals the sender to pause at a safe point until it receives notification from this Actor that it has made sufficient progress.
// To send a message, the sender should call this function on the receiver and pass itself as the first argument.
// A nil first argument is valid, and will prevent backpressure from applying, in cases where an Actor wants to send a message to itself or must receive a message from non-Actor code.
func (a *Inbox) RecvFrom(sender Actor, message func()) {
	if a.enqueue(message) > backpressureThreshold && sender != nil {
		done := make(chan struct{})
		a.enqueue(func() { close(done) })
		sender.RecvFrom(nil, func() { <-done })
	}
}

// SyncExec places a message in the Inbox and returns a channel that will be closed when the Actor finishes handling the message.
// Actors should never, under any circumstances, call SyncExec on another Actor's Inbox and then wait for the channel to close.
// It's meant exclusively as a convenience function for non-Actor code to send messages, and wait for responses, without flooding.
func (a *Inbox) SyncExec(f func()) chan struct{} {
	done := make(chan struct{})
	a.enqueue(func() { f(); close(done) })
	return done
}

// run is executed when a message is placed in an empty Inbox, and launches a worker goroutine.
// The worker goroutine processes messages from the Inbox until empty, and then exits.
func (a *Inbox) run() {
	for {
		head := a.head
		head.msg()
		for {
			a.head = (*queueElem)(atomic.LoadPointer(&head.next))
			if a.head != nil {
				// Move to the next message
				break
			} else if !atomic.CompareAndSwapPointer(&a.tail, unsafe.Pointer(head), nil) {
				// The head is not the tail, but there was no head.next when we checked
				// Somebody must be updating it right now, so try again
				continue
			} else {
				// Head and tail are now both nil, our work here is done, exit
				return
			}
		}
	}
}
