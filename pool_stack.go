package itogami

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// global memory pool for all items used in Pool
var itemPool = sync.Pool{New: func() any { return &directItem{next: nil, value: nil} }}

// Stack implements lock-free freelist based stack
// Credits -> https://github.com/golang-design/lockfree
type Stack struct {
	top unsafe.Pointer
}

// a single item in this stack
type directItem struct {
	next  unsafe.Pointer
	value *slot
}

// NewStack returns a new stack
func NewStack() *Stack {
	return &Stack{}
}

// Pop pops value from the top of the stack
func (s *Stack) Pop() (value *slot) {
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
func (s *Stack) Push(v *slot) {
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
