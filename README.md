# gony

Gony is a small actor model library for Go, inspired by the causal messaging system in the [Pony](https://ponylang.io/) programming language.

Actors send and receive messages in the form of functions of 0 arguments with no return value. These are typically closures that read or modify values only reachable by the actor, and may cause the actor to send messages to other actors.

When an Actor sends a message to some destination Actor that has a much larger queue of work to do, some backpressure is applied. Specifically, the sender will ask the destination to notify it at some point after it handles the sent message, and then the sender will schedule itself to sleep at some later point until it receives a notification. This ensures that any Actor flooded with too much work (relative to other actors) is given time to complete that work before the other Actors send more.

This was written in an attempt to find a way around some issues with deadlockin goroutines in problems that are naturally modeled best as a network of communicating components (where naive use of go channels can either cause deadlocks or force you to drop some messages). The implementation probably isn't particularly efficient, so it's meant more as a proof-of-concept than something anyone would actually use.
