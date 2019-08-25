// Package phony is a small actor model library for Go, inspired by the causal messaging system in the Pony programming language.
// An Actor is a lightweight struct that can be embedded in other structs and send messages to anything with a matching interface.
// Messages are functions of 0 arguments, typically closures, and should not perform blocking operations.
// Message passing is asynchronous, causal, and fast.
// Actors are scheduled to prevent messages queues from growing too large, by pausing at safe breakpoints when an Actor detects that it sent something to another Actor whose inbox is flooded.
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

// An Actor maintans an inbox of messages and processes them 1 at a time.
// Messages are meant to be in the form of non-blocking functions of 0 arguments, often closures.
// The intent is for the Actor struct to be embedded in other structs, where the other fields of the struct are only read or modified by the Actor.
// The basic idea is to write functions for the struct that embeds the actor, which create and pass messages to the Actor when called.
// It is up to the user to ensure that memory is used safely, and that messages do not contain blocking operations.
// An Actor must not be copied after first use.
type Actor struct {
	head *queueElem     // Used carefully to avoid needing atomics
	tail unsafe.Pointer // *queueElem, accessed atomically
}

// IActor is the interface for Actors, based on their ability to enqueue a message from another IActor.
// It's meant so that structs which embed an Actor can be used as such, rather than trying to depend on the concrete Actor type.
type IActor interface {
	EnqueueFrom(IActor, func())
}

// enqueue puts a message on the Actor's inbox queue and returns the number of messages that have been enqueued since the inbox was last empty.
// If the inbox was empty, then the actor was not already running, so enqueue starts it.
func (a *Actor) enqueue(f func()) uint {
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

// EnqueueFrom adds a message to the actor's queue and, if the queue is flooded, signals the sender to pause at a safe point until it receives notification from this Actor that it has made sufficient progress.
// To send a message, the sender should call this function on the receiver and pass itself as the first argument.
// A non-Actor that wants to send a message to an Actor, or an Actor that wants to enqueue a message to itself, may pass nil as the first argument to avoid applying backpressure.
func (a *Actor) EnqueueFrom(sender IActor, message func()) {
	if a.enqueue(message) > backpressureThreshold && sender != nil {
		done := make(chan struct{})
		a.enqueue(func() { close(done) })
		sender.EnqueueFrom(nil, func() { <-done })
	}
}

// SyncExec sends a message to an Actor using EnqueueFrom, and returns a channel that will be closed when the actor finishes handling the message.
// Actors should never, under any circumstances, call SyncExec on another Actor and then wait for the channel to close.
// It's meant exclusively as a convenience function for non-Actor code to send messages, and wait for responses, without flooding.
func (a *Actor) SyncExec(f func()) chan struct{} {
	done := make(chan struct{})
	a.enqueue(func() { f(); close(done) })
	return done
}

// run is executed when a message is enqueued in an empty inbox, and launches a worker goroutine.
// The worker goroutine processes messages from the inbox until empty, and then exits.
func (a *Actor) run() {
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
