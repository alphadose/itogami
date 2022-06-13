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
	_p1      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	maxSize  uint64
	_p2      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	// using a stack keeps cpu caches warm based on FILO property (in theory atleast)
	workerQ *Stack
	_p3     [cacheLinePadSize - unsafe.Sizeof(&Stack{})]byte
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
			go p.loopQ(unsafe.Pointer(&slot{task: task}))
			return
		} else {
			mcall(gosched_m)
		}
	}
}

func (p *Pool) loopQ(s unsafe.Pointer) {
	(*slot)(s).threadPtr = GetG()
	for {
		(*slot)(s).task()
		p.workerQ.Push(s)
		mcall(fast_park)
	}
}
