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
	_p3      [cacheLinePadSize - unsafe.Sizeof(func() {})]byte
	workerQ  *Stack
	_p4      [cacheLinePadSize - unsafe.Sizeof(&Stack{})]byte
}

func NewPoolWithFunc[T any](size uint64, task func(T)) *PoolWithFunc[T] {
	return &PoolWithFunc[T]{workerQ: NewStack(), maxSize: size, task: task}
}

func (p *PoolWithFunc[T]) Invoke(value T) {
	var s unsafe.Pointer = nil
	for {
		if s = p.workerQ.Pop(); s != nil {
			(*dataPoint[T])(s).data = value
			safe_ready((*dataPoint[T])(s).threadPtr)
			return
		} else if atomic.LoadUint64(&p.currSize) < p.maxSize {
			atomic.AddUint64(&p.currSize, 1)
			go p.loopQ(unsafe.Pointer(&dataPoint[T]{data: value}))
			return
		} else {
			mcall(gosched_m)
		}
	}
}

func (p *PoolWithFunc[T]) loopQ(d unsafe.Pointer) {
	(*dataPoint[T])(d).threadPtr = GetG()
	for {
		p.task((*dataPoint[T])(d).data)
		p.workerQ.Push(d)
		mcall(fast_park)
	}
}
