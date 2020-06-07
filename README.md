# Phony

[![Go Report Card](https://goreportcard.com/badge/github.com/Arceliar/phony)](https://goreportcard.com/report/github.com/Arceliar/phony)

[godoc](https://godoc.org/github.com/Arceliar/phony)

Phony is a *very* minimal actor model library for Go, inspired by the causal messaging system in the [Pony](https://ponylang.io/) programming language. This was written in a weekend as an exercise/test, to demonstrate how easily the Actor model can be implemented in Go, rather than as something intended for real-world use. Note that these are Actors running in the local process (as in Pony), not in other processes or on other machines (as in [Erlang](https://www.erlang.org/)).

Phony was written in response to a few places where, in my opinion, idiomatic Go leaves a lot to be desired:

1. Cyclic networks of goroutines that communicate over channels can deadlock, so you end up needing to either drop messages or write some manual buffering or scheduling logic (which is often error prone). Or you can rewrite your code to have no cycles, but sometimes the problem at hand is best modeled with the cycles. I don't really like any of these options. Go makes concurrency and communication *easy*, but combining them isn't *safe*.
2. Goroutines that wait for work from a channel can leak if not signaled to shut down properly, and that shutdown mechanism needs to be manually implemented in most cases. Sometimes it's as easy as ranging over a channel and defering a close, other times it can be a lot more complicated. It's annoying that Go is garbage collected, but it's killer features (goroutines and channels) still need manual management to avoid leaks.
3. I'm tired of writing infinite for loops over select statements. The code is not reusable and resists composition. Lets say I have some type which normally has a worker goroutine associated with it, sitting in a for loop over a select statement. If I want to embed that type in a new struct, which includes any additional channels that must be selected on, I need to rewrite the entire select loop. There's no mechanism to say "and also add this one behavior" without enumerating the full list of behaviors I want from my worker. This is depressing in light of how nicely things behave when a struct anonymously embeds a type, where fields and functions compose beautifully.

## Features

1. Small implementation, only around 100 lines of code, excluding tests and examples. It depends only on a couple of commonly used standard library packages.
2. `Actor`s are extremely lightweight. On `x86_64`, an actor only takes up 24 bytes for their `Inbox` plus 24 bytes per message. While not running, an `Actor` has no associated goroutines, and it can be garbage collected just like any other object when it is no longer needed, even for cycles of `Actor`s.
3. Asynchronous message passing between `Actor`s. Unlike networks go goroutines communicating over channels, sending messages between `Actor`s cannot deadlock.
4. Unbounded Inbox sizes are kept small in practice through backpressure and scheduling. `Actor`s that send to an overworked recipient will pause at a safe point in the future, and wait until signaled that the recipient has caught up. A paused `Actor` also has no associated goroutine or stack.

## Benchmarks

```
goos: linux
goarch: amd64
pkg: github.com/Arceliar/phony
BenchmarkBlock-4             	 1000000	      1666 ns/op	     128 B/op	       2 allocs/op
BenchmarkAct-4               	 3350524	       347 ns/op	       0 B/op	       0 allocs/op
BenchmarkActFromSelf-4       	 3659487	       349 ns/op	      52 B/op	       3 allocs/op
BenchmarkActFromNil-4        	19720999	        57.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkActFromMany-4       	 4216896	       278 ns/op	       2 B/op	       0 allocs/op
BenchmarkActFromManyNil-4    	 4626250	       249 ns/op	       0 B/op	       0 allocs/op
BenchmarkPingPong-4          	 1855938	       592 ns/op	       0 B/op	       0 allocs/op
BenchmarkChannelMany-4       	 2298198	       470 ns/op	       0 B/op	       0 allocs/op
BenchmarkChannel-4           	 1384917	       867 ns/op	       0 B/op	       0 allocs/op
BenchmarkBufferedChannel-4   	15319149	        72.0 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/Arceliar/phony	16.584s
```

If you're here then presumably you can read Go, so I'd recommend just checking the code to see exactly what the benchmarks are testing.
