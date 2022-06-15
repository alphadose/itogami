package itogami

import (
	"sync/atomic"
	"unsafe"
)

type dataPoint[T any] struct {
	threadPtr unsafe.Pointer
	data      T
}

type PoolWithFunc[T any] struct {
	currSize uint64
	_p1      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	maxSize  uint64
	_p2      [cacheLinePadSize - unsafe.Sizeof(uint64(0))]byte
	task     func(T)
	_p3      [cacheLinePadSize - unsafe.Sizeof(func(T) {})]byte
	*StackFunc
	_p4 [cacheLinePadSize - unsafe.Sizeof(&Stack{})]byte
}

func NewPoolWithFunc[T any](size uint64, task func(T)) *PoolWithFunc[T] {
	return &PoolWithFunc[T]{StackFunc: NewStackFunc(), maxSize: size, task: task}
}

func (p *PoolWithFunc[T]) Invoke(value T) {
	var s unsafe.Pointer
	for {
		if s = p.Pop(); s != nil {
			(*dataPoint[T])(s).data = value
			safe_ready((*dataPoint[T])(s).threadPtr)
			return
		} else if atomic.AddUint64(&p.currSize, 1) <= p.maxSize {
			go p.loopQ(unsafe.Pointer(&dataPoint[T]{data: value}))
			return
		} else {
			atomic.AddUint64(&p.currSize, uint64SubtractionConstant)
			mcall(gosched_m)
		}
	}
}

func (p *PoolWithFunc[T]) loopQ(d unsafe.Pointer) {
	(*dataPoint[T])(d).threadPtr = GetG()
	for {
		p.task((*dataPoint[T])(d).data)
		p.Push(d)
		mcall(fast_park)
	}
}
