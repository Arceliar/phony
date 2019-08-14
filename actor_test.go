package phony

import (
	"testing"
	"unsafe"
)

func TestActorSize(t *testing.T) {
	var a Actor
	var q queueElem
	t.Logf("Actor size: %d, message size: %d", unsafe.Sizeof(a), unsafe.Sizeof(q))
}

func TestSyncExec(t *testing.T) {
	var a Actor
	var results []int
	for idx := 0; idx < 1024; idx++ {
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

func TestSendMessageTo(t *testing.T) {
	var a Actor
	done := make(chan struct{})
	var results []int
	a.SyncExec(func() {
		for idx := 0; idx < 1024; idx++ {
			n := idx // Because idx gets mutated in place
			a.SendMessageTo(&a, func() {
				results = append(results, n)
			})
		}
	})
	a.SendMessageTo(&a, func() { close(done) })
	<-done
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
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

func BenchmarkSyncExec(b *testing.B) {
	var a Actor
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.SyncExec(f)
	}
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

func BenchmarkBufferedChannel(b *testing.B) {
	done := make(chan struct{})
	go func() {
		ch := make(chan func(), b.N)
		f := func() {}
		for i := 0; i < b.N; i++ {
			ch <- f
		}
		close(ch)
		for f := range ch {
			f()
		}
		close(done)
	}()
	<-done
}
