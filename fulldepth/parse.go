package fulldepth

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ParseDepthHeader parses the 12-byte header
// Bytes 0-1: Message length (int16, little endian)
// Byte 2: Response code (41=bid, 51=ask, 50=error)
// Byte 3: Exchange segment
// Bytes 4-7: Security ID (int32, little endian)
// Bytes 8-11: Number of rows (int32, little endian)
func ParseDepthHeader(data []byte) (*DepthHeader, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("insufficient data for header: got %d bytes, need 12", len(data))
	}

	header := &DepthHeader{
		MessageLength:   int16(binary.LittleEndian.Uint16(data[0:2])),
		ResponseCode:    data[2],
		ExchangeSegment: data[3],
		SecurityID:      int32(binary.LittleEndian.Uint32(data[4:8])),
		NumRows:         int32(binary.LittleEndian.Uint32(data[8:12])),
	}

	return header, nil
}

// ParseDepthData parses depth data (bid or ask) from a binary message
// Returns the parsed data and remaining bytes for the next message
func ParseDepthData(data []byte, depthLevel DepthLevel) (*DepthData, []byte, error) {
	if len(data) < 12 {
		return nil, nil, fmt.Errorf("insufficient data for depth: got %d bytes, need at least 12", len(data))
	}

	header, err := ParseDepthHeader(data)
	if err != nil {
		return nil, nil, err
	}

	// Check message length
	if int(header.MessageLength) > len(data) {
		return nil, nil, fmt.Errorf("incomplete message: header says %d bytes, got %d", header.MessageLength, len(data))
	}

	// Handle error/disconnect message
	if header.ResponseCode == FeedCodeDisconnect {
		return nil, nil, fmt.Errorf("server disconnection: exchange=%d, security=%d", header.ExchangeSegment, header.SecurityID)
	}

	// Validate response code
	isBid := false
	switch header.ResponseCode {
	case FeedCodeBid:
		isBid = true
	case FeedCodeAsk:
		isBid = false
	default:
		return nil, nil, fmt.Errorf("unknown response code: %d", header.ResponseCode)
	}

	// Determine number of rows to parse
	numRows := int(header.NumRows)
	if depthLevel == Depth20 && numRows > 20 {
		numRows = 20
	} else if depthLevel == Depth200 && numRows > 200 {
		numRows = 200
	}

	// Parse depth entries
	// Each entry is 16 bytes: float64 (price) + uint32 (quantity) + uint32 (orders)
	entrySize := 16
	entries := make([]DepthEntry, 0, numRows)
	offset := 12 // Skip header

	for i := 0; i < numRows; i++ {
		if offset+entrySize > len(data) {
			break
		}

		entry := DepthEntry{
			Price:    bytesToFloat64(data[offset : offset+8]),
			Quantity: int32(binary.LittleEndian.Uint32(data[offset+8 : offset+12])),
			Orders:   int32(binary.LittleEndian.Uint32(data[offset+12 : offset+16])),
		}
		entries = append(entries, entry)
		offset += entrySize
	}

	depthData := &DepthData{
		Header:  *header,
		IsBid:   isBid,
		Entries: entries,
	}

	// Return remaining data for next message
	msgEnd := int(header.MessageLength)
	if msgEnd > len(data) {
		msgEnd = len(data)
	}

	var remaining []byte
	if msgEnd < len(data) {
		remaining = data[msgEnd:]
	}

	return depthData, remaining, nil
}

// bytesToFloat64 converts 8 bytes to float64 (little endian)
func bytesToFloat64(b []byte) float64 {
	if len(b) < 8 {
		return 0
	}
	bits := binary.LittleEndian.Uint64(b)
	return math.Float64frombits(bits)
}
