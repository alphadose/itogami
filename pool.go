package itogami

import (
	"sync/atomic"
	"unsafe"
)

// a single slot for a worker in the thread pool
type slot struct {
	threadPtr unsafe.Pointer
	task      func()
}

// Pool represents the thread-pool for performing any kind of task ( type -> func() {} )
type Pool struct {
	currSize uint64
	_p1      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	maxSize  uint64
	_p2      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	// using a stack keeps cpu caches warm based on FILO property
	*Stack
	_p3 [cacheLinePadSize - unsafe.Sizeof(&Stack{})]byte
}

// NewPool returns a new thread pool
func NewPool(size uint64) *Pool {
	return &Pool{Stack: NewStack(), maxSize: size}
}

// Submit submits a new task to the pool
// it first tries to use already existing goroutines
// if all existing goroutines are present, it tries to add a new goroutine to the pool if the pool capacity is not exceeded
// in case the pool capacity exits, this function yields the processor to other goroutines and loops again for finding available workers
func (p *Pool) Submit(task func()) {
	var s *slot
	for {
		if s = p.Pop(); s != nil {
			s.task = task
			safe_ready(s.threadPtr)
			return
		} else if atomic.AddUint64(&p.currSize, 1) <= p.maxSize {
			s = &slot{task: task}
			go p.loopQ(s)
			return
		} else {
			atomic.AddUint64(&p.currSize, uint64SubtractionConstant)
			mcall(gosched_m)
		}
	}
}

// loopQ is the looping function for every worker goroutine
func (p *Pool) loopQ(s *slot) {
	// store self goroutine pointer
	s.threadPtr = GetG()
	for {
		// exec task
		s.task()
		// notify availability by pushing self reference into stack
		p.Push(s)
		// park and wait for call
		mcall(fast_park)
	}
}
