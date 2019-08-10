package gony

import (
	"testing"
)

func TestActOrder(t *testing.T) {
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

func TestSendTo(t *testing.T) {
	var a Actor
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

func BenchmarkSendTo(b *testing.B) {
	var a Actor
	f := func() {}
	for i := 0; i < b.N; i++ {
		a.SendTo(&a, f)
	}
	done := make(chan struct{})
	a.SendTo(&a, func() { close(done) })
	<-done
}
