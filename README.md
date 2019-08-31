# Phony

[![Go Report Card](https://goreportcard.com/badge/github.com/Arceliar/phony)](https://goreportcard.com/report/github.com/Arceliar/phony)

[godoc](https://godoc.org/github.com/Arceliar/phony)

Phony is a *very* minimal actor model library for Go, inspired by the causal messaging system in the [Pony](https://ponylang.io/) programming language. This was written in a weekend as an exercise/test, to demonstrate how easily the Actor model can be implemented in Go, rather than as something intended for real-world use. Note that these are Actors running in the local process (as in Pony), not in other processes or on other machines (as in [Erlang](https://www.erlang.org/)).

Phony was written in response to a few places where, in my opinion, idiomatic Go leaves a lot to be desired:

1. Cyclic networks of goroutines that communicate over channels can deadlock, so you end up needing to either drop messages or write some manual buffering or scheduling logic (which is often error prone). Or you can rewrite your code to have no cycles, but sometimes the problem at hand is best modeled with the cycles. I don't really like any of these options. Go makes concurrency and communication *easy*, but combining them isn't *safe*.
2. Goroutines that wait for work from a channel can leak if not signaled to shut down properly, and that shutdown mechanism needs to be manually implemented in most cases. Sometimes it's as easy as ranging over a channel and defering a close, other times it can be a lot more complicated. It's annoying that Go is garbage collected, but it's killer features (goroutines and channels) still need manual management to avoid leaks.
3. I'm tired of writing infinite for loops over select statements. The code is not reusable and resists composition. Lets say I have some type which normally has a worker goroutine associated with it, sitting in a for loop over a select statement. If I want to embed that type in a new struct, which includes any additional channels that must be selected on, I need to rewrite the entire select loop. There's no mechanism to say "and also add this one behavior" without enumerating the full list of behaviors I want from my worker. This is depressing in light of how nicely things behave when a struct anonymously embeds a type, where fields and functions compose beautifully.

## Features

1. Small implementation, only about 61 lines of code, excluding tests and examples. It depends only on a couple of commonly used standard library packages.
2. Actors are extremely lightweight, only 24 bytes (on x86_64) when their Inbox is empty. In that state, an Actor has no associated goroutines, and it can be garbage collected just like any other struct, even for cycles of Actors.
3. Asynchronous message passing between Actors. Unlike networks go goroutines communicating over channels, sending messages between Actors cannot deadlock.
4. Unbounded Inbox sizes are kept small in practice through backpressure and scheduling. Actors that send to an overworked recipient will pause at a safe point in the future, and wait until signaled that the recipient has caught up.

## Benchmarks

```
goos: linux
goarch: amd64
pkg: github.com/Arceliar/phony
BenchmarkBlock-4             	 1000000	      1335 ns/op
BenchmarkAct-4               	 5000000	       270 ns/op
BenchmarkActFromNil-4        	20000000	        81.4 ns/op
BenchmarkChannel-4           	 2000000	       846 ns/op
BenchmarkBufferedChannel-4   	20000000	        69.5 ns/op
PASS
ok  	github.com/Arceliar/phony	8.677s
```

If you're here then presumably you can read Go, so I'd recommend just checking the code to see exactly what the benchmarks are testing.
