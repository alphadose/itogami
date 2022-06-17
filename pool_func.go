package itogami

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// a single slot for a worker in PoolWithFunc
type slotFunc[T any] struct {
	threadPtr unsafe.Pointer
	data      T
}

// PoolWithFunc is used for spawning workers for a single pre-defined function with myriad inputs
// useful for throughput bound cases
// has lower memory usage and allocs per op than the default Pool
//  ( type -> func(T) {} ) where T is a generic parameter
type PoolWithFunc[T any] struct {
	currSize uint64
	_p1      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	maxSize  uint64
	_p2      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	task     func(T)
	_p3      [cacheLinePadSize - unsafe.Sizeof(func(T) {})]byte
	top      unsafe.Pointer
	_p4      [cacheLinePadSize - unsafe.Sizeof(unsafe.Pointer(nil))]byte
	free     func(any)
	_p5      [cacheLinePadSize - unsafe.Sizeof(func() {})]byte
	alloc    func() any
	_p6      [cacheLinePadSize - unsafe.Sizeof(func() {})]byte
}

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
	next  unsafe.Pointer
	value *slotFunc[T]
}

// pop pops value from the top of the stack
func (self *PoolWithFunc[T]) pop() (value *slotFunc[T]) {
	var top, next unsafe.Pointer
	for {
		top = atomic.LoadPointer(&self.top)
		if top == nil {
			return
		}
		next = atomic.LoadPointer(&(*dataItem[T])(top).next)
		if atomic.CompareAndSwapPointer(&self.top, top, next) {
			value = (*dataItem[T])(top).value
			(*dataItem[T])(top).next, (*dataItem[T])(top).value = nil, nil
			self.free((*dataItem[T])(top))
			return
		}
	}
}

// push pushes a value on top of the stack
func (self *PoolWithFunc[T]) push(v *slotFunc[T]) {
	var (
		top  unsafe.Pointer
		item = self.alloc().(*dataItem[T])
	)
	item.value = v
	for {
		top = atomic.LoadPointer(&self.top)
		item.next = top
		if atomic.CompareAndSwapPointer(&self.top, top, unsafe.Pointer(item)) {
			return
		}
	}
}
