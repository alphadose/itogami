package itogami

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

var itemPool = sync.Pool{New: func() any { return &directItem{next: nil, value: nil} }}

// Stack implements lock-free freelist based stack.
type Stack struct {
	top unsafe.Pointer
}

type directItem struct {
	next  unsafe.Pointer
	value unsafe.Pointer
}

// NewStack creates a new lock-free queue.
func NewStack() *Stack {
	return &Stack{}
}

// Pop pops value from the top of the stack.
func (s *Stack) Pop() (value unsafe.Pointer) {
	// var top, next unsafe.Pointer
	for {
		top := atomic.LoadPointer(&s.top)
		if top == nil {
			return
		}
		next := atomic.LoadPointer(&(*directItem)(top).next)
		if atomic.CompareAndSwapPointer(&s.top, top, next) {
			value = (*directItem)(top).value
			(*directItem)(top).next, (*directItem)(top).value = nil, nil
			// item.next, item.value = nil, nil
			itemPool.Put((*directItem)(top))
			return
		}
	}
}

// Push pushes a value on top of the stack.
func (s *Stack) Push(v unsafe.Pointer) {
	item := itemPool.Get().(*directItem)
	item.value = v
	// item.next, item.value = nil, v
	var top unsafe.Pointer
	for {
		top = atomic.LoadPointer(&s.top)
		item.next = top
		if atomic.CompareAndSwapPointer(&s.top, top, unsafe.Pointer(item)) {
			return
		}
	}
}
