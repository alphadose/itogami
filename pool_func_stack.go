package itogami

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// global memory pool for all items used in PoolWithFunc
var dataPool = sync.Pool{New: func() any { return &dataItem{next: nil, value: nil} }}

// StackFunc is for pool with func
type StackFunc struct {
	top unsafe.Pointer
}

type dataItem struct {
	next  unsafe.Pointer
	value unsafe.Pointer
}

// NewStack returns a new stack
func NewStackFunc() *StackFunc {
	return &StackFunc{}
}

// Pop pops value from the top of the stack
func (s *StackFunc) Pop() (value unsafe.Pointer) {
	var top, next unsafe.Pointer
	for {
		top = atomic.LoadPointer(&s.top)
		if top == nil {
			return
		}
		next = atomic.LoadPointer(&(*dataItem)(top).next)
		if atomic.CompareAndSwapPointer(&s.top, top, next) {
			value = (*dataItem)(top).value
			(*dataItem)(top).next, (*dataItem)(top).value = nil, nil
			dataPool.Put((*dataItem)(top))
			return
		}
	}
}

// Push pushes a value on top of the stack
func (s *StackFunc) Push(v unsafe.Pointer) {
	var (
		top  unsafe.Pointer
		item = dataPool.Get().(*dataItem)
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
