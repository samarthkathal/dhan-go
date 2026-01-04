package benchmarks

import (
	"testing"

	"github.com/samarthkathal/dhan-go/pool"
)

// BenchmarkBufferPoolGetSmall benchmarks getting a small buffer from the pool
func BenchmarkBufferPoolGetSmall(b *testing.B) {
	bp := pool.NewBufferPool()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := bp.Get(512)
		bp.Put(buf)
	}
}

// BenchmarkBufferPoolGetMedium benchmarks getting a medium buffer from the pool
func BenchmarkBufferPoolGetMedium(b *testing.B) {
	bp := pool.NewBufferPool()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := bp.Get(4096)
		bp.Put(buf)
	}
}

// BenchmarkBufferPoolGetLarge benchmarks getting a large buffer from the pool
func BenchmarkBufferPoolGetLarge(b *testing.B) {
	bp := pool.NewBufferPool()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := bp.Get(32768)
		bp.Put(buf)
	}
}

// BenchmarkBufferPoolGetPut benchmarks getting and putting multiple buffers
func BenchmarkBufferPoolGetPut(b *testing.B) {
	bp := pool.NewBufferPool()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf1 := bp.Get(512)
		buf2 := bp.Get(4096)
		buf3 := bp.Get(32768)
		bp.Put(buf1)
		bp.Put(buf2)
		bp.Put(buf3)
	}
}

// BenchmarkBufferPoolParallel benchmarks parallel buffer pool access
func BenchmarkBufferPoolParallel(b *testing.B) {
	bp := pool.NewBufferPool()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get(1024)
			// Simulate some work with the buffer
			buf[0] = 'x'
			bp.Put(buf)
		}
	})
}

// BenchmarkBufferPoolWrite benchmarks writing to pooled buffers
func BenchmarkBufferPoolWrite(b *testing.B) {
	bp := pool.NewBufferPool()
	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := bp.Get(512)
		copy(buf, data)
		bp.Put(buf)
	}
}

// BenchmarkBufferPoolWriteParallel benchmarks parallel writing to pooled buffers
func BenchmarkBufferPoolWriteParallel(b *testing.B) {
	bp := pool.NewBufferPool()
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get(1024)
			copy(buf, data)
			bp.Put(buf)
		}
	})
}

// BenchmarkNoPool compares performance without buffer pooling
func BenchmarkNoPool(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := make([]byte, 1024)
		copy(buf, data)
		_ = buf
	}
}

// BenchmarkWithPool compares performance with buffer pooling
func BenchmarkWithPool(b *testing.B) {
	bp := pool.NewBufferPool()
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := bp.Get(1024)
		copy(buf, data)
		bp.Put(buf)
	}
}

// BenchmarkGlobalBufferPool benchmarks the global buffer pool functions
func BenchmarkGlobalBufferPool(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.GetBuffer(1024)
		pool.PutBuffer(buf)
	}
}
