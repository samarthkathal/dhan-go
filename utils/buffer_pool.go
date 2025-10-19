package utils

import (
	"sync"
)

// BufferPool is a thread-safe pool of byte slices for reuse
// Reduces GC pressure by reusing buffers for WebSocket operations
type BufferPool struct {
	small  *sync.Pool // For messages < 1KB (ticker data)
	medium *sync.Pool // For messages 1-10KB (quote data)
	large  *sync.Pool // For messages > 10KB (full depth data)
}

const (
	// Buffer size tiers
	smallBufferSize  = 1024      // 1KB - for ticker packets (16 bytes)
	mediumBufferSize = 10 * 1024 // 10KB - for quote packets (50 bytes)
	largeBufferSize  = 64 * 1024 // 64KB - for full packets (150 bytes) + overhead
)

// NewBufferPool creates a new buffer pool with three size tiers
func NewBufferPool() *BufferPool {
	return &BufferPool{
		small: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, smallBufferSize)
				return &b
			},
		},
		medium: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, mediumBufferSize)
				return &b
			},
		},
		large: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, largeBufferSize)
				return &b
			},
		},
	}
}

// Get returns a buffer from the pool based on the requested size
// The returned buffer may be larger than requested
func (bp *BufferPool) Get(size int) []byte {
	var pool *sync.Pool

	switch {
	case size <= smallBufferSize:
		pool = bp.small
	case size <= mediumBufferSize:
		pool = bp.medium
	default:
		pool = bp.large
	}

	bufPtr := pool.Get().(*[]byte)
	buf := *bufPtr

	// Return slice of requested size (or full capacity if smaller)
	if size > cap(buf) {
		// Requested size exceeds pool capacity, allocate new buffer
		return make([]byte, size)
	}

	return buf[:size]
}

// Put returns a buffer to the pool for reuse
// Only buffers from the pool should be returned
func (bp *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)
	var pool *sync.Pool

	switch {
	case capacity == smallBufferSize:
		pool = bp.small
	case capacity == mediumBufferSize:
		pool = bp.medium
	case capacity == largeBufferSize:
		pool = bp.large
	default:
		// Buffer is not from pool (custom allocation), don't return it
		return
	}

	// Reset slice to full capacity before returning to pool
	buf = buf[:capacity]
	pool.Put(&buf)
}

// globalBufferPool is the default buffer pool used by the library
var globalBufferPool = NewBufferPool()

// GetBuffer returns a buffer from the global pool
func GetBuffer(size int) []byte {
	return globalBufferPool.Get(size)
}

// PutBuffer returns a buffer to the global pool
func PutBuffer(buf []byte) {
	globalBufferPool.Put(buf)
}
