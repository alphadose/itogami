package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alphadose/itogami"
)

const runTimes uint32 = 1000

var sum uint32

func myFunc(i uint32) {
	atomic.AddUint32(&sum, i)
	fmt.Printf("run with %d\n", i)
}

func demoFunc() {
	time.Sleep(10 * time.Millisecond)
	println("Hello World")
}

func examplePool() {
	var wg sync.WaitGroup

	syncCalculateSum := func() {
		demoFunc()
		wg.Done()
	}
	for i := uint32(0); i < runTimes; i++ {
		wg.Add(1)
		// Submit task to the pool
		itogami.Submit(syncCalculateSum)
	}
	wg.Wait()
	println("finished all tasks")
}

func examplePoolWithFunc() {
	var wg sync.WaitGroup
	// Use the pool with a pre-defined function
	pool := itogami.NewPoolWithFunc(10, func(i uint32) {
		myFunc(i)
		wg.Done()
	})
	for i := uint32(0); i < runTimes; i++ {
		wg.Add(1)
		// Invoke the function with a value
		pool.Invoke(i)
	}
	wg.Wait()
	fmt.Printf("finish all tasks, result is %d\n", sum)
}

func main() {
	examplePool()
	examplePoolWithFunc()
}
