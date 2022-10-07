package itogami

import (
	"sync/atomic"
	"unsafe"
)

// Pool represents the thread-pool for performing any kind of task ( type -> func() {} )
type Pool struct {
	// using a stack keeps cpu caches warm based on FILO property
	top atomic.Pointer[node]
	_   [cacheLinePadSize - unsafe.Sizeof(atomic.Pointer[node]{})]byte
}

// NewPool returns a new thread pool
func NewPool(size uint64) *Pool {
	return new(Pool)
}

// Submit submits a new task to the pool
// it first tries to use already parked goroutines from the stack if any
// if there are no available worker goroutines, it tries to add a
// new goroutine to the pool if the pool capacity is not exceeded
// in case the pool capacity hit its maximum limit, this function yields the processor to other
// goroutines and loops again for finding available workers
func (p *Pool) Submit(task func()) {
	var s *node
	for {
		if s = p.pop(); s != nil {
			s.task = task
			safe_ready(s.threadPtr)
			return
		} else {
			s = &node{task: task}
			go p.loopQ(s)
			return
		}
	}
}

// loopQ is the looping function for every worker goroutine
func (p *Pool) loopQ(s *node) {
	// store self goroutine pointer
	s.threadPtr = GetG()
	for {
		// exec task
		s.task()
		// notify availability by pushing self reference into stack
		p.push(s)
		// park and wait for call
		mcall(fast_park)
	}
}

// internal lock-free stack implementation for parking and waking up goroutines
// Credits -> https://github.com/golang-design/lockfree

// a single node in this stack
type node struct {
	next      atomic.Pointer[node]
	threadPtr unsafe.Pointer
	task      func()
}

// pop pops value from the top of the stack
func (p *Pool) pop() *node {
	var top, next *node
	for {
		top = p.top.Load()
		if top == nil {
			return nil
		}
		next = top.next.Load()
		if p.top.CompareAndSwap(top, next) {
			top.next.Store(nil)
			return top
		}
	}
}

// push pushes a value on top of the stack
func (p *Pool) push(item *node) {
	var top *node
	for {
		top = p.top.Load()
		item.next.Store(top)
		if p.top.CompareAndSwap(top, item) {
			return
		}
	}
}
