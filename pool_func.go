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
	maxSize  uint64
	task     func(T)
	workerQ  *List
}

func NewPoolWithFunc[T any](size uint64, task func(T)) *PoolWithFunc[T] {
	return &PoolWithFunc[T]{workerQ: NewList(), maxSize: size, task: task}
}

func (p *PoolWithFunc[T]) Invoke(value T) {
	var s unsafe.Pointer = nil
	for {
		if s = p.workerQ.Dequeue(); s != nil {
			(*dataPoint[T])(s).data = value
			safe_ready((*dataPoint[T])(s).threadPtr)
			return
		} else if atomic.LoadUint64(&p.currSize) < p.maxSize {
			atomic.AddUint64(&p.currSize, 1)
			go p.loopQ(&dataPoint[T]{data: value})
			return
		} else {
			mcall(gosched_m)
		}
	}
}

func (p *PoolWithFunc[T]) loopQ(d *dataPoint[T]) {
	d.threadPtr = GetG()
	for {
		p.task(d.data)
		p.workerQ.Enqueue(unsafe.Pointer(d))
		mcall(fast_park)
	}
}
