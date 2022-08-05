package itogami

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type (
	// a single slot for a worker in PoolWithFunc
	slotFunc[T any] struct {
		threadPtr unsafe.Pointer
		data      T
	}

	// PoolWithFunc is used for spawning workers for a single pre-defined function with myriad inputs
	// useful for throughput bound cases
	// has lower memory usage and allocs per op than the default Pool
	//
	//	( type -> func(T) {} ) where T is a generic parameter
	PoolWithFunc[T any] struct {
		currSize uint64
		_p1      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
		maxSize  uint64
		alloc    func() any
		free     func(any)
		task     func(T)
		_p2      [cacheLinePadSize - unsafe.Sizeof(uint64(0)) - 3*unsafe.Sizeof(func() {})]byte
		top      atomic.Pointer[dataItem[T]]
		_p3      [cacheLinePadSize - unsafe.Sizeof(atomic.Pointer[dataItem[T]]{})]byte
	}
)

// NewPoolWithFunc returns a new PoolWithFunc
func NewPoolWithFunc[T any](size uint64, task func(T)) *PoolWithFunc[T] {
	dataPool := sync.Pool{New: func() any { return new(dataItem[T]) }}
	return &PoolWithFunc[T]{maxSize: size, task: task, alloc: dataPool.Get, free: dataPool.Put}
}

// Invoke invokes the pre-defined method in PoolWithFunc by assigning the data to an already existing worker
// or spawning a new worker given queue size is in limits
func (self *PoolWithFunc[T]) Invoke(value T) {
	var s *slotFunc[T]
	for {
		if s = self.pop(); s != nil {
			s.data = value
			safe_ready(s.threadPtr)
			return
		} else if atomic.AddUint64(&self.currSize, 1) <= self.maxSize {
			s = &slotFunc[T]{data: value}
			go self.loopQ(s)
			return
		} else {
			atomic.AddUint64(&self.currSize, uint64SubtractionConstant)
			mcall(gosched_m)
		}
	}
}

// represents the infinite loop for a worker goroutine
func (self *PoolWithFunc[T]) loopQ(d *slotFunc[T]) {
	d.threadPtr = GetG()
	for {
		self.task(d.data)
		self.push(d)
		mcall(fast_park)
	}
}

// Stack implementation below for storing goroutine references

// a single node in the stack
type dataItem[T any] struct {
	next  atomic.Pointer[dataItem[T]]
	value *slotFunc[T]
}

// pop pops value from the top of the stack
func (self *PoolWithFunc[T]) pop() (value *slotFunc[T]) {
	var top, next *dataItem[T]
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
			self.free(top)
			return
		}
	}
}

// push pushes a value on top of the stack
func (self *PoolWithFunc[T]) push(v *slotFunc[T]) {
	var (
		top  *dataItem[T]
		item = self.alloc().(*dataItem[T])
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
