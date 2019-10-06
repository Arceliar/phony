package phony

import (
	"runtime"
	"sync"
	"testing"
	"unsafe"
)

func TestInboxSize(t *testing.T) {
	var a Inbox
	var q queueElem
	t.Logf("Inbox size: %d, message size: %d", unsafe.Sizeof(a), unsafe.Sizeof(q))
}

func TestBlock(t *testing.T) {
	var a Inbox
	var results []int
	for idx := 0; idx < 1024; idx++ {
		n := idx // Because idx gets mutated in place
		Block(&a, func() {
			results = append(results, n)
		})
	}
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func TestAct(t *testing.T) {
	var a Inbox
	var results []int
	Block(&a, func() {
		for idx := 0; idx < 1024; idx++ {
			n := idx // Because idx gets mutated in place
			a.Act(&a, func() {
				results = append(results, n)
			})
		}
	})
	Block(&a, func() {})
	for idx, n := range results {
		if n != idx {
			t.Errorf("value %d != index %d", n, idx)
		}
	}
}

func BenchmarkBlock(b *testing.B) {
	var a Inbox
	for i := 0; i < b.N; i++ {
		Block(&a, func() {})
	}
}

func BenchmarkAct(b *testing.B) {
	var a, s Inbox
	done := make(chan struct{})
	idx := 0
	var f func()
	f = func() {
		if idx < b.N {
			idx++
			a.Act(&s, func() {})
			s.Act(nil, f)
		} else {
			a.Act(&s, func() { close(done) })
		}
	}
	s.Act(nil, f)
	<-done
}

func BenchmarkActFromNil(b *testing.B) {
	var a Inbox
	done := make(chan struct{})
	idx := 0
	var f func()
	f = func() {
		if idx < b.N {
			idx++
			a.Act(nil, f)
		} else {
			close(done)
		}
	}
	a.Act(nil, f)
	<-done
}

func BenchmarkActFromMany(b *testing.B) {
	var s Inbox
	count := runtime.GOMAXPROCS(0)
	var group sync.WaitGroup
	for idx := 0; idx < count; idx++ {
		msgs := b.N / count
		if idx == 0 {
			msgs = b.N - (count-1)*msgs
		}
		var a Inbox
		jdx := 0
		var f func()
		f = func() {
			if jdx < msgs {
				jdx++
				a.Act(&s, func() {})
				s.Act(nil, f)
			} else {
				a.Act(&s, func() { group.Done() })
			}
		}
		group.Add(1)
		a.Act(nil, f)
	}
	group.Wait()
}

func BenchmarkActFromManyNil(b *testing.B) {
	var s Inbox
	count := runtime.GOMAXPROCS(0)
	var group sync.WaitGroup
	for idx := 0; idx < count; idx++ {
		msgs := b.N / count
		if idx == 0 {
			msgs = b.N - (count-1)*msgs
		}
		var a Inbox
		jdx := 0
		var f func()
		f = func() {
			if jdx < msgs {
				jdx++
				a.Act(nil, func() {})
				s.Act(nil, f)
			} else {
				a.Act(nil, func() { group.Done() })
			}
		}
		group.Add(1)
		a.Act(nil, f)
	}
	group.Wait()
}

func BenchmarkPingPong(b *testing.B) {
	var pinger, ponger Inbox
	done := make(chan struct{})
	idx := 0
	var ping, pong func()
	ping = func() {
		if idx < b.N {
			idx++
			ponger.Act(&pinger, pong)
		} else {
			close(done)
		}
	}
	pong = func() {
		if idx < b.N {
			idx++
			pinger.Act(&ponger, ping)
		} else {
			close(done)
		}
	}
	pinger.Act(nil, ping)
	<-done
}

func BenchmarkChannelMany(b *testing.B) {
	done := make(chan struct{})
	ch := make(chan func())
	go func() {
		for f := range ch {
			f()
		}
		close(done)
	}()
	var group sync.WaitGroup
	count := runtime.GOMAXPROCS(0)
	for idx := 0; idx < count; idx++ {
		msgs := b.N / count
		if idx == 0 {
			msgs = b.N - (count-1)*msgs
		}
		group.Add(1)
		go func() {
			f := func() {}
			for jdx := 0; jdx < msgs; jdx++ {
				ch <- f
			}
			group.Done()
		}()
	}
	group.Wait()
	close(ch)
	<-done
}

func BenchmarkChannel(b *testing.B) {
	done := make(chan struct{})
	ch := make(chan func())
	go func() {
		for f := range ch {
			ch <- f
		}
		close(done)
	}()
	f := func() {}
	for i := 0; i < b.N; i++ {
		ch <- f
		g := <-ch
		g()
	}
	close(ch)
	<-done
}

func BenchmarkBufferedChannel(b *testing.B) {
	ch := make(chan func(), 1)
	f := func() {}
	for i := 0; i < b.N; i++ {
		ch <- f
		g := <-ch
		g()
	}
}
