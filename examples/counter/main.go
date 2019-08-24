package main

import (
	"fmt"

	"github.com/Arceliar/phony"
)

// Structs can embed the Actor type to fulfill the IActor interface.
type printer struct {
	phony.Actor
}

// Functions can be defined to send messages to an Actor from another Actor.
func (p *printer) Println(from phony.IActor, msg ...interface{}) {
	p.EnqueueFrom(from, func() { fmt.Println(msg...) })
}

// It's useful to embed an Actor in a struct whose fields the Actor is responsible for.
type counter struct {
	phony.Actor
	count   int
	printer *printer
}

// An EnqueueFrom nil is useful for asking an actor to do something from non-actor code.
func (c *counter) Increment() {
	c.EnqueueFrom(nil, func() { c.count++ })
}

// A SyncExec function returns a channel that will be closed after the actor finishes handling the message
// This can be used to interrogate an Actor from an outside goroutine.
// Note that Actors shouldn't use this on eachother, since it blocks, it's just meant for convenience when interfacing with outside code.
func (c *counter) Get(n *int) {
	<-c.SyncExec(func() { *n = c.count })
}

// Print sends a message to the counter, telling to to call c.printer.Println
// Calling Println sends a message to the printer, telling it to print
// So message sends become function calls.
func (c *counter) Print() {
	c.EnqueueFrom(c, func() {
		c.printer.Println(c, "The count is:", c.count)
	})
}

func main() {
	c := &counter{printer: new(printer)} // Create an actor
	for idx := 0; idx < 10; idx++ {
		c.Increment() // Ask the actor to do some work
		c.Print()     // And ask it to send a message to another actor, which handles them asynchronously
	}
	var n int
	c.Get(&n)                         // Inspect the actor's internal state
	fmt.Println("Value from Get:", n) // This likely prints before the Print() lines above have finished -- actors work asynchronously.
	<-c.printer.SyncExec(func() {})   // Wait for an actor to handle a message, in this case just to finish printing
	fmt.Println("Exiting")
}
