package gony

import (
	"sync"
	"testing"
	"unsafe"
)

func TestAct(t *testing.T) {
	var a Actor
	done := make(chan struct{})
	var results []int
	for idx := 0; idx < 16; idx++ {
		n := idx // Because idx gets mutated in place
		a.Act(func() {
			results = append(results, n)
		})
	}
	a.Act(func() { close(done) })
	<-done
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func BenchmarkAct(b *testing.B) {
	var a Actor
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.Act(f)
	}
	done := make(chan struct{})
	a.Act(func() { close(done) })
	<-done
}

func TestSendMessageTo(t *testing.T) {
	var a Actor
	done := make(chan struct{})
	var results []int
	for idx := 0; idx < 16; idx++ {
		n := idx // Becuase idx gets mutated in place
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

////////////////////////////////////////////////////////////////////////////////

func TestActOrder_old(t *testing.T) {
	var a Actor_old
	done := make(chan struct{})
	var results []int
	for idx := 0; idx < 16; idx++ {
		n := idx // Because idx gets mutated in place
		a.Act(func() {
			results = append(results, n)
		})
	}
	a.Act(func() { close(done) })
	<-done
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func BenchmarkAct_old(b *testing.B) {
	var a Actor_old
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.Act(f)
	}
	done := make(chan struct{})
	a.Act(func() { close(done) })
	<-done
}

func TestSendTo_old(t *testing.T) {
	var a Actor_old
	done := make(chan struct{})
	var results []int
	for idx := 0; idx < 16; idx++ {
		n := idx // Becuase idx gets mutated in place
		a.SendTo(&a, func() {
			results = append(results, n)
		})
	}
	a.SendTo(&a, func() { close(done) })
	<-done
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func BenchmarkSendTo_old(b *testing.B) {
	var a Actor_old
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.SendTo(&a, f)
	}
	done := make(chan struct{})
	a.SendTo(&a, func() { close(done) })
	<-done
}

func TestBackpressure_old(t *testing.T) {
	var a, b, o Actor_old
	// o is just some dummy observer we use to put messages on a/b queues
	ch_a := make(chan struct{})
	ch_b := make(chan struct{})
	// Block b to make sure pressure builds
	o.SendTo(&b, func() { <-ch_b })
	// Have A spam messages to B
	for idx := 0; idx < 1024; idx++ {
		o.SendTo(&a, func() {
			// The inner call happens in A's worker
			a.SendTo(&b, func() {})
		})
	}
	o.SendTo(&a, func() { close(ch_a) })
	// ch_a should now be blocked until ch_b closes
	close(ch_b)
	<-ch_a
}

////////////////////////////////////////////////////////////////////////////////

func BenchmarkMutex(b *testing.B) {
	var mutex sync.Mutex
	for i := 0; i < b.N; i++ {
		mutex.Lock()
		mutex.Unlock()
	}
}

func BenchmarkChannel(b *testing.B) {
	in := make(chan chan struct{}, 1024)
	go func() {
		for ch := range in {
			close(ch)
		}
	}()
	var ch chan struct{}
	for i := 0; i < b.N; i++ {
		ch = make(chan struct{})
		in <- ch
	}
	close(in)
	<-ch
}

func TestUnsafePointer(t *testing.T) {
	var p unsafe.Pointer
	if p != nil {
		t.Errorf("Expected nil, got %v", p)
	}
}
