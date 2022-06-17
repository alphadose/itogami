package itogami

import (
	"sync"
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
	top unsafe.Pointer
	_p3 [cacheLinePadSize - unsafe.Sizeof(unsafe.Pointer(nil))]byte
}

// NewPool returns a new thread pool
func NewPool(size uint64) *Pool {
	return &Pool{maxSize: size}
}

// Submit submits a new task to the pool
// it first tries to use already existing goroutines
// if all existing goroutines are present, it tries to add a new goroutine to the pool if the pool capacity is not exceeded
// in case the pool capacity exits, this function yields the processor to other goroutines and loops again for finding available workers
func (p *Pool) Submit(task func()) {
	var s *slot
	for {
		if s = p.pop(); s != nil {
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
		p.push(s)
		// park and wait for call
		mcall(fast_park)
	}
}

// global memory pool for all items used in Pool
var itemPool = sync.Pool{New: func() any { return &directItem{next: nil, value: nil} }}

// internal lock-free stack implementation for parking and waking up goroutines
// Credits -> https://github.com/golang-design/lockfree

// a single item in this stack
type directItem struct {
	next  unsafe.Pointer
	value *slot
}

// Pop pops value from the top of the stack
func (s *Pool) pop() (value *slot) {
	var top, next unsafe.Pointer
	for {
		top = atomic.LoadPointer(&s.top)
		if top == nil {
			return
		}
		next = atomic.LoadPointer(&(*directItem)(top).next)
		if atomic.CompareAndSwapPointer(&s.top, top, next) {
			value = (*directItem)(top).value
			(*directItem)(top).next, (*directItem)(top).value = nil, nil
			itemPool.Put((*directItem)(top))
			return
		}
	}
}

// Push pushes a value on top of the stack
func (s *Pool) push(v *slot) {
	var (
		top  unsafe.Pointer
		item = itemPool.Get().(*directItem)
	)
	item.value = v
	for {
		top = atomic.LoadPointer(&s.top)
		item.next = top
		if atomic.CompareAndSwapPointer(&s.top, top, unsafe.Pointer(item)) {
			return
		}
	}
}
