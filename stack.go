package itogami

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

var itemPool = sync.Pool{New: func() any { return unsafe.Pointer(&directItem{next: nil, value: nil}) }}

// Stack implements lock-free freelist based stack
type Stack struct {
	top unsafe.Pointer
}

type directItem struct {
	next  unsafe.Pointer
	value unsafe.Pointer
}

// NewStack returns a new stack
func NewStack() *Stack {
	return &Stack{}
}

// Pop pops value from the top of the stack
func (s *Stack) Pop() (value unsafe.Pointer) {
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
			itemPool.Put(top)
			return
		}
	}
}

// Push pushes a value on top of the stack
func (s *Stack) Push(v unsafe.Pointer) {
	var (
		top  unsafe.Pointer
		item = itemPool.Get().(unsafe.Pointer)
	)
	(*directItem)(item).value = v
	for {
		top = atomic.LoadPointer(&s.top)
		(*directItem)(item).next = top
		if atomic.CompareAndSwapPointer(&s.top, top, item) {
			return
		}
	}
}
