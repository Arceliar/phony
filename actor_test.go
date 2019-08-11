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
	for i := 0; i < b.N; i++ {
		a.SendMessageTo(&a, f)
	}
	done := make(chan struct{})
	a.SendMessageTo(&a, func() { close(done) })
	<-done
}

func BenchmarkBackpressure(b *testing.B) {
	var a0, a1, a2 Actor // 3 actors passing messages in a ring
	done := make(chan struct{})
	a0.SyncExec(func() {
		for idx := 0; idx < b.N; idx++ {
			a0.SendMessageTo(&a1,
				func() {
					a1.SendMessageTo(&a2,
						func() {
							a2.SendMessageTo(&a0, func() {})
						})
				})
		}
		a0.SendMessageTo(&a1,
			func() {
				a1.SendMessageTo(&a2,
					func() {
						a2.SendMessageTo(&a0, func() { close(done) })
					})
			})
	})
	<-done
}

func BenchmarkEnqueue(b *testing.B) {
	var a Actor
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.Enqueue(f)
	}
	done := make(chan struct{})
	a.Enqueue(func() { close(done) })
	<-done // Wait for the worker to finish
}

func BenchmarkEnqueueDelayRunning(b *testing.B) {
	var a Actor
	pause := make(chan struct{})
	a.Enqueue(func() { <-pause }) // Prevent the actor from running
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.Enqueue(f)
	}
	done := make(chan struct{})
	a.Enqueue(func() { close(done) })
	close(pause) // Let the actor do its work
	<-done       // Wait for the worker to finish
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
