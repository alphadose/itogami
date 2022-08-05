package itogami

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// a single slot for a worker in Pool
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
	top atomic.Pointer[node]
	_p3 [cacheLinePadSize - unsafe.Sizeof(atomic.Pointer[node]{})]byte
}

// NewPool returns a new thread pool
func NewPool(size uint64) *Pool {
	return &Pool{maxSize: size}
}

// Submit submits a new task to the pool
// it first tries to use already parked goroutines from the stack if any
// if there are no available worker goroutines, it tries to add a
// new goroutine to the pool if the pool capacity is not exceeded
// in case the pool capacity hit its maximum limit, this function yields the processor to other
// goroutines and loops again for finding available workers
func (self *Pool) Submit(task func()) {
	var s *slot
	for {
		if s = self.pop(); s != nil {
			s.task = task
			safe_ready(s.threadPtr)
			return
		} else if atomic.AddUint64(&self.currSize, 1) <= self.maxSize {
			s = &slot{task: task}
			go self.loopQ(s)
			return
		} else {
			atomic.AddUint64(&self.currSize, uint64SubtractionConstant)
			mcall(gosched_m)
		}
	}
}

// loopQ is the looping function for every worker goroutine
func (self *Pool) loopQ(s *slot) {
	// store self goroutine pointer
	s.threadPtr = GetG()
	for {
		// exec task
		s.task()
		// notify availability by pushing self reference into stack
		self.push(s)
		// park and wait for call
		mcall(fast_park)
	}
}

// global memory pool for all items used in Pool
var (
	itemPool  = sync.Pool{New: func() any { return new(node) }}
	itemAlloc = itemPool.Get
	itemFree  = itemPool.Put
)

// internal lock-free stack implementation for parking and waking up goroutines
// Credits -> https://github.com/golang-design/lockfree

// a single node in this stack
type node struct {
	next  atomic.Pointer[node]
	value *slot
}

// pop pops value from the top of the stack
func (self *Pool) pop() (value *slot) {
	var top, next *node
	for {
		top = self.top.Load()
		if top == nil {
			return
		}
		next = top.next.Load()
		if self.top.CompareAndSwap(top, next) {
			value = top.value
			top.value = nil
			top.next.Store(nil)
			itemFree(top)
			return
		}
	}
}

// push pushes a value on top of the stack
func (self *Pool) push(v *slot) {
	var (
		top  *node
		item = itemAlloc().(*node)
	)
	item.value = v
	for {
		top = self.top.Load()
		item.next.Store(top)
		if self.top.CompareAndSwap(top, item) {
			return
		}
	}
}
