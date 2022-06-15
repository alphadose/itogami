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
	// using a stack keeps cpu caches warm based on FILO property
	*Stack
	_p3 [cacheLinePadSize - unsafe.Sizeof(&Stack{})]byte
}

func NewPool(size uint64) *Pool {
	return &Pool{Stack: NewStack(), maxSize: size}
}

func (p *Pool) Submit(task func()) {
	var s unsafe.Pointer
	for {
		if s = p.Pop(); s != nil {
			(*slot)(s).task = task
			safe_ready((*slot)(s).threadPtr)
			return
		} else if atomic.AddUint64(&p.currSize, 1) <= p.maxSize {
			go p.loopQ(unsafe.Pointer(&slot{task: task}))
			return
		} else {
			atomic.AddUint64(&p.currSize, uint64SubtractionConstant)
			mcall(gosched_m)
		}
	}
}

func (p *Pool) loopQ(s unsafe.Pointer) {
	(*slot)(s).threadPtr = GetG()
	for {
		(*slot)(s).task()
		p.Push(s)
		mcall(fast_park)
	}
}
