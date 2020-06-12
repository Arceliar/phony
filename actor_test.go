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

func TestPanicAct(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	var a Inbox
	a.Act(nil, nil)
}

func TestPanicBlockActor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	Block(nil, nil)
}

func TestPanicBlockAction(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	var a Inbox
	Block(&a, nil)
}

func BenchmarkLoopActor(b *testing.B) {
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

func BenchmarkLoopChannel(b *testing.B) {
	ch := make(chan func(), 1)
	defer close(ch)
	go func() {
		for f := range ch {
			f()
		}
	}()
	done := make(chan struct{})
	idx := 0
	var f func()
	f = func() {
		if idx < b.N {
			idx++
			ch <- f
		} else {
			close(done)
		}
	}
	ch <- f
	<-done
}

func BenchmarkSendActor(b *testing.B) {
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

func BenchmarkSendChannel(b *testing.B) {
	done := make(chan struct{})
	ch := make(chan func())
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

func BenchmarkRequestResponseActor(b *testing.B) {
	var pinger, ponger Inbox
	done := make(chan struct{})
	idx := 0
	var ping, pong func()
	ping = func() {
		if idx < b.N {
			idx++
			ponger.Act(&pinger, pong)
			pinger.Act(nil, ping) // loop asynchronously
		} else {
			ponger.Act(&pinger, func() {
				pinger.Act(nil, func() {
					close(done)
				})
			})
		}
	}
	pong = func() {
		pinger.Act(nil, func() {}) // send a response without backpressure
	}
	pinger.Act(nil, ping)
	<-done
}

func BenchmarkRequestResponseChannel(b *testing.B) {
	done := make(chan struct{})
	toPing := make(chan func(), 1)
	toPong := make(chan func(), 1)
	defer close(toPing)
	defer close(toPong)
	var ping func()
	var pong func()
	ping = func() {
		for idx := 0; idx < b.N; idx++ {
			toPong <- pong
			f := <-toPing
			f()
		}
		toPong <- func() {
			toPing <- func() {
				close(done)
			}
		}
	}
	pong = func() {
		toPing <- func() {}
	}
	go func() {
		for f := range toPing {
			f()
		}
	}()
	go func() {
		for f := range toPong {
			f()
		}
	}()
	toPing <- ping
	<-done
}

func BenchmarkBlock(b *testing.B) {
	var a Inbox
	for i := 0; i < b.N; i++ {
		Block(&a, func() {})
	}
}
