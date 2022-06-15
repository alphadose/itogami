package test

import (
	"sync"
	"testing"
	"time"

	"github.com/alphadose/itogami"
	"github.com/panjf2000/ants/v2"
)

const (
	RunTimes           = 1e6
	BenchParam         = 10
	PoolSize           = 2e5
	DefaultExpiredTime = 10 * time.Second
)

func demoFunc() {
	time.Sleep(time.Duration(BenchParam) * time.Millisecond)
}

func demoPoolFunc(args interface{}) {
	n := args.(int)
	time.Sleep(time.Duration(n) * time.Millisecond)
}
func BenchmarkGoroutines(b *testing.B) {
	var wg sync.WaitGroup

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			go func() {
				demoFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkAntsPool(b *testing.B) {
	var wg sync.WaitGroup
	p, _ := ants.NewPool(PoolSize, ants.WithExpiryDuration(DefaultExpiredTime))
	defer p.Release()

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			_ = p.Submit(func() {
				demoFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkItogamiPool(b *testing.B) {
	var wg sync.WaitGroup
	p := itogami.NewPool(PoolSize)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			p.Submit(func() {
				demoFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkAntsPooWithFunc(b *testing.B) {
	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(PoolSize, demoPoolFunc, ants.WithExpiryDuration(DefaultExpiredTime))
	defer p.Release()

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			p.Invoke(1)
			wg.Done()
		}
		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkItogamiPoolWithFunc(b *testing.B) {
	var wg sync.WaitGroup
	p := itogami.NewPoolWithFunc(PoolSize, demoPoolFunc)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			p.Invoke(1)
			wg.Done()
		}
		wg.Wait()
	}
	b.StopTimer()
}
