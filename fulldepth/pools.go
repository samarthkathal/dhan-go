package fulldepth

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

// bytesToFloat64 converts 8 bytes to float64 (little endian)
func bytesToFloat64(b []byte) float64 {
	if len(b) < 8 {
		return 0
	}
	bits := binary.LittleEndian.Uint64(b)
	return math.Float64frombits(bits)
}

// Pools for struct reuse to reduce GC pressure in high-throughput scenarios
var (
	headerPool = sync.Pool{
		New: func() interface{} { return &DepthHeader{} },
	}
	depthDataPool = sync.Pool{
		New: func() interface{} { return &DepthData{} },
	}
	fullDepthPool = sync.Pool{
		New: func() interface{} { return &FullDepthData{} },
	}
	// Pre-allocated entry slices for common sizes
	entries20Pool = sync.Pool{
		New: func() interface{} {
			s := make([]DepthEntry, 0, 20)
			return &s
		},
	}
	entries200Pool = sync.Pool{
		New: func() interface{} {
			s := make([]DepthEntry, 0, 200)
			return &s
		},
	}
)

// acquireDepthHeader gets a header from the pool
func acquireDepthHeader() *DepthHeader {
	return headerPool.Get().(*DepthHeader)
}

// releaseDepthHeader returns a header to the pool
func releaseDepthHeader(h *DepthHeader) {
	if h == nil {
		return
	}
	*h = DepthHeader{}
	headerPool.Put(h)
}

// acquireDepthData gets a depth data from the pool
func acquireDepthData() *DepthData {
	return depthDataPool.Get().(*DepthData)
}

// releaseDepthData returns a depth data to the pool
func releaseDepthData(d *DepthData) {
	if d == nil {
		return
	}
	// Don't put the slice back - let it be GC'd or reused separately
	d.Header = DepthHeader{}
	d.IsBid = false
	d.Entries = nil
	depthDataPool.Put(d)
}

// acquireFullDepthData gets a full depth data from the pool
func acquireFullDepthData() *FullDepthData {
	return fullDepthPool.Get().(*FullDepthData)
}

// releaseFullDepthData returns a full depth data to the pool
func releaseFullDepthData(f *FullDepthData) {
	if f == nil {
		return
	}
	f.ExchangeSegment = 0
	f.SecurityID = 0
	f.Bids = nil
	f.Asks = nil
	fullDepthPool.Put(f)
}

// acquireEntries20 gets a pre-allocated 20-entry slice from the pool
func acquireEntries20() []DepthEntry {
	ptr := entries20Pool.Get().(*[]DepthEntry)
	return (*ptr)[:0] // Reset length, keep capacity
}

// releaseEntries20 returns a 20-entry slice to the pool
func releaseEntries20(entries []DepthEntry) {
	if cap(entries) < 20 {
		return // Don't pool undersized slices
	}
	entries = entries[:0]
	entries20Pool.Put(&entries)
}

// acquireEntries200 gets a pre-allocated 200-entry slice from the pool
func acquireEntries200() []DepthEntry {
	ptr := entries200Pool.Get().(*[]DepthEntry)
	return (*ptr)[:0]
}

// releaseEntries200 returns a 200-entry slice to the pool
func releaseEntries200(entries []DepthEntry) {
	if cap(entries) < 200 {
		return
	}
	entries = entries[:0]
	entries200Pool.Put(&entries)
}

// parseDepthDataPooled parses depth data using pooled allocation
// Caller MUST call releaseDepthData when done with the data
func parseDepthDataPooled(data []byte, depthLevel DepthLevel) (*DepthData, []byte, error) {
	if len(data) < 12 {
		return nil, nil, fmt.Errorf("insufficient data for depth: got %d bytes, need at least 12", len(data))
	}

	// Parse header inline to avoid allocation
	msgLen := int16(binary.LittleEndian.Uint16(data[0:2]))
	respCode := data[2]
	exchSeg := data[3]
	secID := int32(binary.LittleEndian.Uint32(data[4:8]))
	numRows := int32(binary.LittleEndian.Uint32(data[8:12]))

	// Check message length
	if int(msgLen) > len(data) {
		return nil, nil, fmt.Errorf("incomplete message: header says %d bytes, got %d", msgLen, len(data))
	}

	// Handle error/disconnect message
	if respCode == FeedCodeDisconnect {
		return nil, nil, fmt.Errorf("server disconnection: exchange=%d, security=%d", exchSeg, secID)
	}

	// Validate response code
	isBid := false
	switch respCode {
	case FeedCodeBid:
		isBid = true
	case FeedCodeAsk:
		isBid = false
	default:
		return nil, nil, fmt.Errorf("unknown response code: %d", respCode)
	}

	// Determine number of rows to parse
	rowCount := int(numRows)
	if depthLevel == Depth20 && rowCount > 20 {
		rowCount = 20
	} else if depthLevel == Depth200 && rowCount > 200 {
		rowCount = 200
	}

	// Get pooled entry slice
	var entries []DepthEntry
	if depthLevel == Depth20 {
		entries = acquireEntries20()
	} else {
		entries = acquireEntries200()
	}

	// Parse depth entries
	entrySize := 16
	offset := 12
	for i := 0; i < rowCount; i++ {
		if offset+entrySize > len(data) {
			break
		}
		entries = append(entries, DepthEntry{
			Price:    bytesToFloat64(data[offset : offset+8]),
			Quantity: int32(binary.LittleEndian.Uint32(data[offset+8 : offset+12])),
			Orders:   int32(binary.LittleEndian.Uint32(data[offset+12 : offset+16])),
		})
		offset += entrySize
	}

	// Get pooled depth data struct
	depthData := acquireDepthData()
	depthData.Header.MessageLength = msgLen
	depthData.Header.ResponseCode = respCode
	depthData.Header.ExchangeSegment = exchSeg
	depthData.Header.SecurityID = secID
	depthData.Header.NumRows = numRows
	depthData.IsBid = isBid
	depthData.Entries = entries

	// Return remaining data
	msgEnd := int(msgLen)
	if msgEnd > len(data) {
		msgEnd = len(data)
	}

	var remaining []byte
	if msgEnd < len(data) {
		remaining = data[msgEnd:]
	}

	return depthData, remaining, nil
}

// combineDepthDataPooled combines bid and ask into a FullDepthData using pooled allocation
// Caller MUST call releaseFullDepthData when done
func combineDepthDataPooled(bid, ask *DepthData) *FullDepthData {
	full := acquireFullDepthData()
	if bid != nil {
		full.ExchangeSegment = bid.Header.ExchangeSegment
		full.SecurityID = bid.Header.SecurityID
		full.Bids = bid.Entries
	}
	if ask != nil {
		if bid == nil {
			full.ExchangeSegment = ask.Header.ExchangeSegment
			full.SecurityID = ask.Header.SecurityID
		}
		full.Asks = ask.Entries
	}
	return full
}

// =============================================================================
// SAFE CALLBACK-BASED API (Recommended for library users)
// =============================================================================
// These functions automatically manage pool lifecycle - no manual Release needed.
// The pooled object is only valid within the callback scope.

// WithDepthData parses depth data and calls fn with the result.
// The DepthData is automatically returned to the pool after fn returns.
// Data is only valid during callback. To retain: myDepth := depth.Copy()
func WithDepthData(data []byte, depthLevel DepthLevel, fn func(*DepthData, []byte) error) error {
	depthData, remaining, err := parseDepthDataPooled(data, depthLevel)
	if err != nil {
		return err
	}
	defer releaseDepthData(depthData)
	return fn(depthData, remaining)
}

// WithFullDepthData parses bid and ask data, combines them, and calls fn with the result.
// All pooled objects are automatically returned to the pool after fn returns.
// Data is only valid during callback. To retain: myFull := full.Copy()
func WithFullDepthData(bidData, askData []byte, depthLevel DepthLevel, fn func(*FullDepthData) error) error {
	bid, _, err := parseDepthDataPooled(bidData, depthLevel)
	if err != nil {
		return err
	}
	defer releaseDepthData(bid)

	ask, _, err := parseDepthDataPooled(askData, depthLevel)
	if err != nil {
		return err
	}
	defer releaseDepthData(ask)

	full := combineDepthDataPooled(bid, ask)
	defer releaseFullDepthData(full)

	return fn(full)
}
