package timecache

import (
	"sync"
	"testing"
	"time"
)

func BenchmarkTimeNow(b *testing.B) {
	b.SetParallelism(10000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = time.Now()
		}
	})
}

func BenchmarkTimeNowConcurrent(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/100; j++ {
				_ = time.Now()
			}
		}()
	}
	wg.Wait()
}

func BenchmarkTimeCache(b *testing.B) {
	// timeCache := New(time.Millisecond)
	// Set(timeCache)

	b.SetParallelism(10000)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Now()
		}
	})
}

func BenchmarkTimeCacheConcurrent(b *testing.B) {
	tc := New(time.Microsecond)
	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/100; j++ {
				_ = tc.Now()
			}
		}()
	}
	wg.Wait()
}
