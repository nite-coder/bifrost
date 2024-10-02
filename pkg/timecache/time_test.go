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

func TestTimeCache(t *testing.T) {
	// Test case 1: Check if the returned time is not zero
	tc := New(time.Second)
	now := tc.Now()
	if now.IsZero() {
		t.Errorf("Expected non-zero time, got zero time")
	}

	// Test case 2: Check if the returned time is updated after refresh
	tc = New(time.Second)
	now1 := tc.Now()
	time.Sleep(2 * time.Second)
	now2 := tc.Now()
	if now1.Equal(now2) {
		t.Errorf("Expected different time, got the same time")
	}

	// Test case 3: Check if the returned time is within the interval
	tc = New(time.Second)
	now1 = tc.Now()
	time.Sleep(time.Millisecond)
	now2 = tc.Now()
	if !now1.Add(time.Millisecond).After(now2) {
		t.Errorf("Expected time within interval, got time outside interval")
	}
}
