package phony

import (
	"testing"
	"unsafe"
)

func TestInboxSize(t *testing.T) {
	var a Inbox
	var q queueElem
	t.Logf("Inbox size: %d, message size: %d", unsafe.Sizeof(a), unsafe.Sizeof(q))
}

func TestSyncExec(t *testing.T) {
	var a Inbox
	var results []int
	for idx := 0; idx < 1024; idx++ {
		n := idx // Because idx gets mutated in place
		<-a.SyncExec(func() {
			results = append(results, n)
		})
	}
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func TestRecvFrom(t *testing.T) {
	var a Inbox
	var results []int
	<-a.SyncExec(func() {
		for idx := 0; idx < 1024; idx++ {
			n := idx // Because idx gets mutated in place
			a.RecvFrom(&a, func() {
				results = append(results, n)
			})
		}
	})
	<-a.SyncExec(func() {})
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func BenchmarkSyncExec(b *testing.B) {
	var a Inbox
	for i := 0; i < b.N; i++ {
		<-a.SyncExec(func() {})
	}
}

func BenchmarkEnqueueFrom(b *testing.B) {
	var a0, a1 Inbox
	var count int
	done := make(chan struct{})
	var f func()
	f = func() {
		// Run in a0
		if count < b.N {
			a1.RecvFrom(&a0, func() {})
			count++
			// Continue the loop by sending a message to ourself to run the next iteration.
			// If there's any backpressure from a1, this gives it a chance to apply.
			a0.RecvFrom(nil, f)
		} else {
			a1.RecvFrom(&a0, func() { close(done) })
		}
	}
	a0.RecvFrom(nil, f)
	<-done
}

func BenchmarkEnqueueFromNil(b *testing.B) {
	var a0, a1 Inbox
	done := make(chan struct{})
	a0.RecvFrom(nil, func() {
		for idx := 0; idx < b.N; idx++ {
			// We don't care about backpressure, so we just enqueue the message in a for loop.
			a1.RecvFrom(nil, func() {})
		}
		a1.RecvFrom(nil, func() { close(done) })
	})
	<-done
}

func BenchmarkChannelSyncExec(b *testing.B) {
	ch := make(chan func())
	done := make(chan struct{})
	go func() {
		for f := range ch {
			f()
		}
		close(done)
	}()
	f := func() {}
	for i := 0; i < b.N; i++ {
		d := make(chan struct{})
		ch <- func() { f(); close(d) }
		<-d
	}
	close(ch)
	<-done
}

func BenchmarkChannel(b *testing.B) {
	done := make(chan struct{})
	ch := make(chan func())
	go func() {
		for f := range ch {
			f()
		}
		close(done)
	}()
	go func() {
		f := func() {}
		for i := 0; i < b.N; i++ {
			ch <- f
		}
		close(ch)
	}()
	<-done
}

func BenchmarkBufferedChannel(b *testing.B) {
	done := make(chan struct{})
	ch := make(chan func(), b.N)
	go func() {
		for f := range ch {
			f()
		}
		close(done)
	}()
	go func() {
		f := func() {}
		for i := 0; i < b.N; i++ {
			ch <- f
		}
		close(ch)
	}()
	<-done
}
