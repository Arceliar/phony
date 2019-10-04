//+build !mobile

package phony

// How large a queue can be before backpressure slows down sending to it.
const backpressureThreshold = 255
