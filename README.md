# Phony

[![Go Report Card](https://goreportcard.com/badge/github.com/Arceliar/phony)](https://goreportcard.com/report/github.com/Arceliar/phony)
[![GoDoc](https://godoc.org/github.com/Arceliar/phony?status.svg)](https://godoc.org/github.com/Arceliar/phony)

Phony is a Go implementation of shared memory actor model concurrency, inspired by the [Pony](https://ponylang.io/) programming language, with asynchronous causal message passing and backpressure.

TODO: explain the problems of goroutines+channels and what actors have to offer. a *short* explanation.

## Features

1. Tiny package with less than 100 source lines of code, which depends only on builtin Go primitives and a few commonly used packages from the standard library.
2. Tiny API: There 1 `interface` (`Actor`) with 1 exported function (`Act`), implemented by 1 `type` (`Inbox`). There's also 1 helper function (`Block`) that can be (*carefully*) used to export blocking APIs from `Actor` code.
3. Easy to use. Implementing an `Actor` is as simple as embedding an `Inbox` into the `struct` the `Actor` should "own". Convering a synchronous function with no return argument into an asynchronous one is typically 2 lines of `gofmt`-style code (by wrapping the function body in a call to `Act`).
4. `Actor`s are light weight. On `x86_64`, an idle `Actor`'s empty `Inbox` is 24 bytes, and the overhead per message is 16 bytes (plus whatever the size is of the closure and its captured variables). Idle `Actor`s have no associated goroutine. `Actor`s are garbage collected when no longer reachable -- there are no leaks.
5. Code written in the `Actor` style (where `Actor`s do not call blocking functions) is deadlock-free.

## Benchmarks

```
goos: linux
goarch: amd64
pkg: github.com/Arceliar/phony
BenchmarkLoopActor-4                	18698307	        55.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkLoopChannel-4              	16105111	        72.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkSendActor-4                	 3521646	       325 ns/op	       0 B/op	       0 allocs/op
BenchmarkSendChannel-4              	 2271639	       446 ns/op	       0 B/op	       0 allocs/op
BenchmarkRequestResponseActor-4     	 2788890	       412 ns/op	       0 B/op	       0 allocs/op
BenchmarkRequestResponseChannel-4   	 1218379	       893 ns/op	       0 B/op	       0 allocs/op
BenchmarkBlock-4                    	  776150	      1572 ns/op	     128 B/op	       2 allocs/op
PASS
ok  	github.com/Arceliar/phony	10.295s
```

Obligatory disclaimer: these are micro-benchmarks, not real world applications, so your mileage may vary. I suspect the performance differences between `Actor`s and goroutines+channels will be negligible in most applications.

## Implementation Details

- TODO
