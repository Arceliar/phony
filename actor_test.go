package gony

import (
	"testing"
)

func TestSyncExec(t *testing.T) {
	var a Actor
	var results []int
	for idx := 0; idx < 16; idx++ {
		n := idx // Because idx gets mutated in place
		a.SyncExec(func() {
			results = append(results, n)
		})
	}
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func BenchmarkSyncExec(b *testing.B) {
	var a Actor
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.SyncExec(f)
	}
}

func TestSendMessageTo(t *testing.T) {
	var a Actor
	done := make(chan struct{})
	var results []int
	for idx := 0; idx < 16; idx++ {
		n := idx // Because idx gets mutated in place
		a.SendMessageTo(&a, func() {
			results = append(results, n)
		})
	}
	a.SendMessageTo(&a, func() { close(done) })
	<-done
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func BenchmarkSendMessageTo(b *testing.B) {
	var a Actor
	f := func() {}
	a.SyncExec(func() {
		for i := 0; i < b.N; i++ {
			a.SendMessageTo(&a, f)
		}
	})
	// Wait for the actor to finish
	a.SyncExec(func() {})
}

func BenchmarkBackpressure(b *testing.B) {
	var a0, a1 Actor
	msg := func() {}
	a0.SyncExec(func() {
		for idx := 0; idx < b.N; idx++ {
			a0.SendMessageTo(&a1, msg)
		}
	})
	done := make(chan struct{})
	a0.Enqueue(func() { a0.SendMessageTo(&a1, func() { close(done) }) })
	<-done
}

func BenchmarkEnqueue(b *testing.B) {
	var a Actor
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.Enqueue(f)
	}
	// Wait for the actor to finish
	a.SyncExec(func() {})
}

func BenchmarkEnqueueDelayRunning(b *testing.B) {
	var a Actor
	pause := make(chan struct{})
	a.Enqueue(func() { <-pause }) // Prevent the actor from running
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.Enqueue(f)
	}
	close(pause) // Let the actor do its work
	// Wait for the actor to finish
	a.SyncExec(func() {})
}

func BenchmarkChannelsSync(b *testing.B) {
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
		ch <- f
	}
	close(ch)
	<-done
}

func BenchmarkChannelsAsync(b *testing.B) {
	ch := make(chan func(), b.N)
	done := make(chan struct{})
	go func() {
		for f := range ch {
			f()
		}
		close(done)
	}()
	f := func() {}
	for i := 0; i < b.N; i++ {
		ch <- f
	}
	close(ch)
	<-done
}

func BenchmarkChannelsAsyncDelayRunning(b *testing.B) {
	ch := make(chan func(), b.N)
	done := make(chan struct{})
	go func() {
		<-done
		for f := range ch {
			f()
		}
		close(done)
	}()
	f := func() {}
	for i := 0; i < b.N; i++ {
		ch <- f
	}
	close(ch)
	done <- struct{}{}
	<-done
}
