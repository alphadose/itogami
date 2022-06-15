package itogami

import (
	"sync/atomic"
	"unsafe"
)

// a single point in PoolWithFunc
type dataPoint[T any] struct {
	threadPtr unsafe.Pointer
	data      T
}

// PoolWithFunc is used for spawning workers for a single pre-defined function with myriad inputs
// useful for throughput bound cases
type PoolWithFunc[T any] struct {
	currSize uint64
	_p1      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	maxSize  uint64
	_p2      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	task     func(T)
	_p3      [cacheLinePadSize - unsafe.Sizeof(func(T) {})]byte
	*StackFunc
	_p4 [cacheLinePadSize - unsafe.Sizeof(&StackFunc{})]byte
}

// NewPoolWithFunc returns a new PoolWithFunc
func NewPoolWithFunc[T any](size uint64, task func(T)) *PoolWithFunc[T] {
	return &PoolWithFunc[T]{StackFunc: NewStackFunc(), maxSize: size, task: task}
}

// Invoke invokes the pre-defined method in PoolWithFunc by assigning the data to an already existing worker
// or spawning a new worker given queue size is in limits
func (p *PoolWithFunc[T]) Invoke(value T) {
	var s unsafe.Pointer
	for {
		if s = p.Pop(); s != nil {
			(*dataPoint[T])(s).data = value
			safe_ready((*dataPoint[T])(s).threadPtr)
			return
		} else if atomic.AddUint64(&p.currSize, 1) <= p.maxSize {
			go p.loopQ(&dataPoint[T]{data: value})
			return
		} else {
			atomic.AddUint64(&p.currSize, uint64SubtractionConstant)
			mcall(gosched_m)
		}
	}
}

// represents the infinite loop for a worker goroutine
func (p *PoolWithFunc[T]) loopQ(d *dataPoint[T]) {
	d.threadPtr = GetG()
	for {
		p.task(d.data)
		p.Push(unsafe.Pointer(d))
		mcall(fast_park)
	}
}
