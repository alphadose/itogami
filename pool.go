package itogami

import (
	"sync/atomic"
	"unsafe"
)

type slot struct {
	threadPtr unsafe.Pointer
	task      func()
}

type Pool struct {
	currSize uint64
	maxSize  uint64
	workerQ  *Stack
}

func NewPool(size uint64) *Pool {
	return &Pool{workerQ: NewStack(), maxSize: size}
}

func (p *Pool) Submit(task func()) {
	var s unsafe.Pointer = nil
	for {
		if s = p.workerQ.Pop(); s != nil {
			(*slot)(s).task = task
			safe_ready((*slot)(s).threadPtr)
			return
		} else if atomic.LoadUint64(&p.currSize) < p.maxSize {
			atomic.AddUint64(&p.currSize, 1)
			go p.loopQ(&slot{task: task})
			return
		} else {
			mcall(gosched_m)
		}
	}
}

func (p *Pool) loopQ(s *slot) {
	s.threadPtr = GetG()
	for {
		s.task()
		s.task = nil
		p.workerQ.Push(unsafe.Pointer(s))
		mcall(fast_park)
	}
}
