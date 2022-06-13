package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alphadose/itogami"
)

var sum int32

func myFunc(i int32) {
	atomic.AddInt32(&sum, i)
	fmt.Printf("run with %d\n", i)
}

func demoFunc() {
	time.Sleep(10 * time.Millisecond)
	fmt.Println("Hello World!")
}

func main() {
	runTimes := int32(1000)
	pool := itogami.NewPool(10)
	// Use the common pool.
	var wg sync.WaitGroup
	syncCalculateSum := func() {
		demoFunc()
		wg.Done()
	}
	for i := int32(0); i < runTimes; i++ {
		wg.Add(1)
		pool.Submit(syncCalculateSum)
	}
	wg.Wait()
	fmt.Printf("finish all tasks.\n")

	// Use the pool with a function,
	// set 10 to the capacity of goroutine pool and 1 second for expired duration.
	p := itogami.NewPoolWithFunc(10, func(i int32) {
		myFunc(i)
		wg.Done()
	})
	// Submit tasks one by one.
	for i := int32(0); i < runTimes; i++ {
		wg.Add(1)
		p.Invoke(i)
	}
	wg.Wait()
	fmt.Printf("finish all tasks, result is %d\n", sum)
}
