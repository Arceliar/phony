# Phony

[![Go Report Card](https://goreportcard.com/badge/github.com/Arceliar/phony)](https://goreportcard.com/report/github.com/Arceliar/phony)

Phony is a *very* minimal actor model library for Go, inspired by the causal messaging system in the [Pony](https://ponylang.io/) programming language. This was written in a weekend as an exercise/test, to demonstrate how easily the Actor model can be implemented in Go, rather than as something intended for real-world use.

## Features

1. An extremely small code base consisting of under 100 SLOC, not counting comments/tests/examples, which only depends on Go built-ins and the standard library.
2. The zero value of an Actor is about 24 bytes on x86_64 and is ready-to-use with no initialization. The intent is to embed it in a struct containing whatever state the Actor is meant to manage.
3. Actors with an empty queue have no associated goroutines. Idle actors, including idle cycles of actors, can be garbage collected just like any other struct, with no "poison pill" needed to prevent leaks.
4. Actors send messages asynchronously and have unbounded queue size -- the goal is no deadlocks, ever. Just be sure that you let the outside part of your code block sending work *to* Actors, and not the other way around.
5. Backpressure keeps the memory usage from unbounded queues in check, by causing Actors which send messages a flooded recipient to (eventually) pause message handling until the recipient notifies them that it made progress.
6. It's surprisingly fast, comparable to sending over channels.

## Benchmarks

```
goos: linux
goarch: amd64
pkg: github.com/Arceliar/phony
BenchmarkEnqueue-4           	20000000	       114 ns/op
BenchmarkSyncExec-4          	 1000000	      1346 ns/op
BenchmarkBackpressure-4      	 5000000	       258 ns/op
BenchmarkSendMessageTo-4     	20000000	        94.6 ns/op
BenchmarkChannelSyncExec-4   	 1000000	      1117 ns/op
BenchmarkChannel-4           	 3000000	       429 ns/op
BenchmarkBufferedChannel-4   	20000000	        73.4 ns/op
PASS
ok  	github.com/Arceliar/phony	11.801s
```

In the above benchmarks, `BenchmarkBackpressure` consists of sending an empty function to an actor as fast as possible, which the actor runs before retrieving the next empty function. `BenchmarkChannel` corresponds to the same workflow, but sending those functions over a channel with no buffer (or a very small buffer that easily fills). I consider these to be the most relevant benchmarks, as is models performance under load.

`BenchmarkSendMessageTo` and `BenchmarkBufferedChannel` correspond to cases where the receiving actor does not require backpressure, or the receiving channel is buffered and not full. These correspond most closely to the case where the receiver can do work faster than it is produced.
