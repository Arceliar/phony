package main

import (
	"fmt"

	"github.com/Arceliar/phony"
)

// Structs can embed the Actor type to fulfill the IActor interface.
type printer struct {
	phony.Actor
}

// It's often convenient to put the receiver of a message in the receiver position of a function call, and then supply the sender as an argument.
func (p *printer) Println(from phony.IActor, msg ...interface{}) {
	from.SendMessageTo(p, func() { fmt.Println(msg...) })
}

// It's useful to embed an Actor in a struct whose fields the Actor is responsible for.
type counter struct {
	phony.Actor
	count   int
	printer *printer
}

// An Enqueue function tells the actor to do something at some point in the future.
func (c *counter) Increment() {
	c.Enqueue(func() { c.count++ })
}

// A SyncExec function waits for the actor to finish, and can be used to interrogate an Actor from an outside goroutine. Note that Actors shouldn't use this on eachother, since it blocks, it's just meant for convenience when interfacing with outside code.
func (c *counter) Get(n *int) {
	c.SyncExec(func() { *n = c.count })
}

// Actors can send messages to other Actors.
func (c *counter) Print() {
	c.Enqueue(func() { c.printer.Println(c, "The count is:", c.count) })
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
	fmt.Scanln()                      // Wait long enough for these things to actually happen.
	fmt.Println("Exiting")
}
