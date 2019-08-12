# gony

[![Go Report Card](https://goreportcard.com/badge/github.com/Arceliar/gony)](https://goreportcard.com/report/github.com/Arceliar/gony)

The "gony" package is a *very* minimal actor model library for Go, inspired by the causal messaging system in the [Pony](https://ponylang.io/) programming language. This was written in a weekend as an exercise/test, to demonstrate how easily the Actor model can be implemented in Go, rather than as something intended for real-world use.

## Features

1. An extremely small code base consisting of about 60 SLOC, not counting tests, which only depends on Go built-ins and the `sync` package from the standard library.
2. The zero value of an Actor is about 32 bytes on x86_64 and is ready-to-use with no initialization. The intent is to embed it in a struct containing whatever state the Actor is meant to manage.
3. Actors with an empty queue have no associated goroutines. Idle actors, including idle cycles of actors, can be garbage collected just like any other struct, with no "poison pill" needed to prevent leaks.
4. Actors send messages asynchronously and have unbounded queue size -- the goal is no deadlocks, ever. Just be sure that you let the outside part of your code block sending work *to* Actors, and not the other way around.
5. Backpressure keeps the memory usage from unbounded queues in check, by causing Actors which send messages a flooded recipient to (eventually) pause message handling until the recipient notifies them that it made progress.
6. It's surprisingly fast (see benchmarks below).

## Benchmarks

```
goos: linux
goarch: amd64
pkg: github.com/Arceliar/gony
BenchmarkEnqueue-4                 	10000000	       131 ns/op
BenchmarkSyncExec-4                	 1000000	      1375 ns/op
BenchmarkBackpressure-4            	10000000	       153 ns/op
BenchmarkSendMessageTo-4           	10000000	       133 ns/op
BenchmarkChannelsSyncExec-4        	 1000000	      1099 ns/op
BenchmarkSmallBufferedChannels-4   	 5000000	       363 ns/op
BenchmarkLargeBufferedChannels-4   	10000000	       133 ns/op
PASS
ok  	github.com/Arceliar/gony	10.794s
```

In the above benchmarks, `BenchmarkBackpressure` consists of sending an empty function to an actor as fast as possible, which the actor runs before retrieving the next function. `BenchmarkSmallBufferedChannels` corresponds to the same workflow, but sending those functions over a goroutine with a small (1) buffer. I consider these to be the most relevant benchmarks, as is models performance under load -- it doesn't *really* matter how long things take when they're not under enough load to trigger backpressure or block channels (`BenchmarkSendMessageTo` and `BenchmarkLargeBufferedChannels`), since that implies most time is spent idle and waiting for work.
