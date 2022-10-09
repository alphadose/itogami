package itogami

import (
	"sync/atomic"
	"unsafe"
)

// Pool represents the thread-pool for performing any kind of task ( type -> func() {} )
type Pool struct {
	// using a stack keeps cpu caches warm based on FILO property
	top atomic.Pointer[node]
	_   [cacheLinePadSize - unsafe.Sizeof(atomic.Pointer[node]{})]byte
}

// globally referenced pool object
var pool = Pool{}

// Submit submits a new task to the pool
// tries to re-use existing goroutine if available else it spawns a new goroutine
func Submit(task func()) {
	if s := pop(); s != nil {
		s.task = task           // assign task to existing worker goroutine
		safe_ready(s.threadPtr) // start the goroutine
	} else {
		go loopQ(task) // spawn new worker goroutine
	}
}

// loopQ is the looping function for every worker goroutine
func loopQ(task func()) {
	state := &node{threadPtr: GetG(), task: task}
	for {
		state.task()     // exec task
		push(state)      // notify availability by pushing state reference into stack
		mcall(fast_park) // park and wait for call
	}
}

// internal lock-free stack implementation for parking and waking up goroutines
// Credits -> https://github.com/golang-design/lockfree

// a single node in this stack
type node struct {
	next      atomic.Pointer[node]
	threadPtr unsafe.Pointer
	task      func()
}

// pop pops value from the top of the stack
func pop() *node {
	var top, next *node
	for {
		top = pool.top.Load()
		if top == nil {
			return nil
		}
		next = top.next.Load()
		if pool.top.CompareAndSwap(top, next) {
			top.next.Store(nil)
			return top
		}
	}
}

// push pushes a value on top of the stack
func push(item *node) {
	var top *node
	for {
		top = pool.top.Load()
		item.next.Store(top)
		if pool.top.CompareAndSwap(top, item) {
			return
		}
	}
}
