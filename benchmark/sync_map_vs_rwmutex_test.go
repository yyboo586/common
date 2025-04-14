package benchmark

import (
	"sync"
	"sync/atomic"
	"testing"
)

/*
goos: linux
goarch: amd64
pkg: github.com/yyboo586/common/benchmark
cpu: AMD Ryzen 5 5600G with Radeon Graphics
BenchmarkSyncMap/100%R-1G-12         	886115347	         1.346 ns/op	       0 B/op	       0 allocs/op
BenchmarkSyncMap/100%R-4G-12         	847539974	         1.371 ns/op	       0 B/op	       0 allocs/op
BenchmarkSyncMap/100%R-16G-12        	884639792	         1.348 ns/op	       0 B/op	       0 allocs/op
BenchmarkSyncMap/50%R-4G-12          	29480835	        66.75 ns/op	      53 B/op	       1 allocs/op
BenchmarkSyncMap/20%R-4G-12          	44679933	        34.01 ns/op	      18 B/op	       0 allocs/op
BenchmarkSyncMap/100%W-4G-12         	22813528	       116.3 ns/op	     117 B/op	       2 allocs/op
BenchmarkRWMutexMap/100%R-1G-12      	43950952	        26.43 ns/op	       0 B/op	       0 allocs/op
BenchmarkRWMutexMap/100%R-4G-12      	44863578	        26.68 ns/op	       0 B/op	       0 allocs/op
BenchmarkRWMutexMap/100%R-16G-12     	45175150	        26.44 ns/op	       0 B/op	       0 allocs/op
BenchmarkRWMutexMap/50%R-4G-12       	11841681	       138.8 ns/op	      37 B/op	       0 allocs/op
BenchmarkRWMutexMap/20%R-4G-12       	30601129	        67.27 ns/op	      14 B/op	       0 allocs/op
BenchmarkRWMutexMap/100%W-4G-12      	 5281285	       257.8 ns/op	      84 B/op	       0 allocs/op
PASS
*/

// BenchmarkSyncMap 测试sync.Map在不同并发模式下的性能
func BenchmarkSyncMap(b *testing.B) {
	benchCases := []struct {
		name       string
		goroutines int
		writeRatio float32 // 写操作比例 0.0~1.0
	}{
		{"100%R-1G", 1, 0.0},
		{"100%R-4G", 4, 0.0},
		{"100%R-16G", 16, 0.0},
		{"50%R-4G", 4, 0.5},
		{"20%R-4G", 4, 0.2},
		{"100%W-4G", 4, 1.0},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			var m sync.Map
			var counter int64
			b.ResetTimer()

			b.SetParallelism(bc.goroutines)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if bc.writeRatio > 0 && float32(atomic.AddInt64(&counter, 1)%100) < bc.writeRatio*100 {
						// 写操作
						m.Store(atomic.LoadInt64(&counter), "value")
					} else {
						// 读操作
						m.Load(atomic.LoadInt64(&counter) % 100)
					}
				}
			})
		})
	}
}

// BenchmarkRWMutexMap 测试RWMutex+map在不同并发模式下的性能
func BenchmarkRWMutexMap(b *testing.B) {
	benchCases := []struct {
		name       string
		goroutines int
		writeRatio float32
	}{
		{"100%R-1G", 1, 0.0},
		{"100%R-4G", 4, 0.0},
		{"100%R-16G", 16, 0.0},
		{"50%R-4G", 4, 0.5},
		{"20%R-4G", 4, 0.2},
		{"100%W-4G", 4, 1.0},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			var mu sync.RWMutex
			m := make(map[int64]string)
			var counter int64
			b.ResetTimer()

			b.SetParallelism(bc.goroutines)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					key := atomic.AddInt64(&counter, 1)
					if bc.writeRatio > 0 && float32(key%100) < bc.writeRatio*100 {
						// 写操作
						mu.Lock()
						m[key] = "value"
						mu.Unlock()
					} else {
						// 读操作
						mu.RLock()
						_ = m[key%100]
						mu.RUnlock()
					}
				}
			})
		})
	}
}
