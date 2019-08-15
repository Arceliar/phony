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
	count int
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

// IActor is the interface for Actors, based on their ability to enqueue and send messages.
// It's meant so that structs which embed an Actor can be used as such, rather than trying to depend on the concrete Actor type.
type IActor interface {
	Enqueue(func()) int
	SendMessageTo(IActor, func())
}

// Enqueue puts a message on the Actor's inbox queue and returns the number of messages that have been Enqueued since the inbox was last empty.
// Enqueue is useful for sending a message to an Actor from non-actor code in cases where synchronization is not needed, and it's used internally by SendMessageTo.
func (a *Actor) Enqueue(f func()) int {
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

// SendMessageTo should normally only be called by the Actor itself, never by another Actor or non-actor code.
// It's used to send messages to another Actor, and schedules the current Actor to wait for a signal at a safe breakpoint if it sees that the recipient's inbox is flooded.
func (a *Actor) SendMessageTo(destination IActor, message func()) {
	if destination.Enqueue(message) > backpressureThreshold && destination != a {
		// Tried to send to someone else, with a large queue, so apply some backpressure
		// Sending backpressure to ourself is perfectly safe, but it's pointless extra work that only serves to slow things down even more, so we don't bother
		done := make(chan struct{})
		destination.Enqueue(func() { close(done) })
		a.Enqueue(func() { <-done })
	}
}

// SyncExec sends a message to an Actor using Enqueue, and then waits for it to be handled before returning.
// Actors should never, under any circumstances, call SyncExec.
// It's meant exclusively as a convenience function for non-actor code, Enqueue messages without flooding the Actor's inbox.
func (a *Actor) SyncExec(f func()) {
	done := make(chan struct{})
	a.Enqueue(func() { f(); close(done) })
	<-done
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
			} else {
				if !atomic.CompareAndSwapPointer(&a.tail, unsafe.Pointer(head), nil) {
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
}
