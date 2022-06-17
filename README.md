# Itogami

> The best goroutine pool in terms of performance

Saves a lot of memory and is the fastest among all existing golang thread-pool implementations

Benchmarks to support the above claims [here](#benchmarks)

## Installation

You need Golang [1.18.x](https://go.dev/dl/) or above since this package uses generics

```bash
$ go get github.com/alphadose/itogami@0.2.0
```

## Usage

```go
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
	// Use the common pool
	pool := itogami.NewPool(10)

	syncCalculateSum := func() {
		demoFunc()
		wg.Done()
	}
	for i := uint32(0); i < runTimes; i++ {
		wg.Add(1)
		// Submit task to the pool
		pool.Submit(syncCalculateSum)
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
```

## Benchmarks

Benchmarking was performed against existing golang threadpool implementations [Ants](https://github.com/panjf2000/ants) and [Gamma-Zero-Worker-Pool](https://github.com/gammazero/workerpool) and unlimited goroutines

Thread pool size -> 50k

CPU -> M1, arm64, 8 cores, 3.2 GHz

OS -> darwin

Results were computed from [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) of 30 cases
```
name                   time/op
UnlimitedGoroutines-8   291ms ± 2%
AntsPool-8              512ms ± 6%
GammaZeroPool-8         713ms ±10%
ItogamiPool-8           319ms ± 1%

name                   alloc/op
UnlimitedGoroutines-8  96.2MB ± 0%
AntsPool-8             21.8MB ± 9%
GammaZeroPool-8        18.8MB ± 1%
ItogamiPool-8          25.8MB ± 3%

name                   allocs/op
UnlimitedGoroutines-8   2.00M ± 0%
AntsPool-8              1.09M ± 3%
GammaZeroPool-8         1.05M ± 0%
ItogamiPool-8           1.05M ± 0%
```

The following conclusions can be drawn from the above results:-

1. Itogami is the fastest among all threadpool implementations and slower only than unlimited goroutines
2. Itogami has the least `allocs/op` and hence the memory usage scales really well with high load
3. The memory used per operation is in the acceptable range of other threadpools and drastically lower than unlimited goroutines
4. The tolerance (± %) for Itogami is quite low for all 3 metrics indicating that the algorithm is quite stable overall


Benchmarking code available [here](https://github.com/alphadose/go-threadpool-benchmarks)
