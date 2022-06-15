package test

import (
	"sync"
	"testing"
	"time"

	"github.com/alphadose/itogami"
	"github.com/panjf2000/ants/v2"
)

var wg1, wg2, wg3 sync.WaitGroup

const sleepDuration uint8 = 10

func antsFunc(args any) {
	time.Sleep(time.Duration(args.(uint8)) * time.Millisecond)
	wg2.Done()
}

func itoFunc(args uint8) {
	time.Sleep(time.Duration(args) * time.Millisecond)
	wg3.Done()
}

func BenchmarkAntsPooWithFunc(b *testing.B) {
	p, _ := ants.NewPoolWithFunc(PoolSize, antsFunc, ants.WithExpiryDuration(DefaultExpiredTime))
	defer p.Release()

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg2.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			p.Invoke(sleepDuration)
		}
		wg2.Wait()
	}
	b.StopTimer()
}

func BenchmarkItogamiPoolWithFunc(b *testing.B) {
	p := itogami.NewPoolWithFunc(PoolSize, itoFunc)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg3.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			p.Invoke(sleepDuration)
		}
		wg3.Wait()
	}
	b.StopTimer()
}
