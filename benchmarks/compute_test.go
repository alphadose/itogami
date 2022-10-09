package test

import (
	"sync"
	"testing"
	"time"
	_ "unsafe"

	"github.com/alphadose/itogami"
)

const epochs = 1e3

func doCopyStack(a, b int) int {
	if b < 100 {
		time.Sleep(time.Microsecond)
		return doCopyStack(0, b+1)
	}
	return 0
}

func demoFunc() {
	doCopyStack(0, 0)
}

func BenchmarkGolangScheduler(b *testing.B) {
	var wg sync.WaitGroup
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(epochs)
		for j := 0; j < epochs; j++ {
			go func() {
				demoFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkItogamiScheduler(b *testing.B) {
	var wg sync.WaitGroup
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(epochs)
		for j := 0; j < epochs; j++ {
			itogami.Submit(func() {
				demoFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}
