package dhan

import "errors"

// Common errors
var (
	// ErrNotConnected is returned when attempting an operation on a disconnected client
	ErrNotConnected = errors.New("client not connected")

	// ErrAlreadyConnected is returned when attempting to connect an already connected client
	ErrAlreadyConnected = errors.New("client already connected")

	// ErrInvalidAccessToken is returned when the access token is empty or invalid
	ErrInvalidAccessToken = errors.New("invalid access token")

	// ErrMaxConnectionsReached is returned when the maximum number of connections is reached
	ErrMaxConnectionsReached = errors.New("maximum connections reached")

	// ErrMaxInstrumentsReached is returned when trying to subscribe to too many instruments
	ErrMaxInstrumentsReached = errors.New("maximum instruments per connection reached")

	// ErrConnectionClosed is returned when trying to use a closed connection
	ErrConnectionClosed = errors.New("connection closed")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timeout")

	// ErrInvalidInstrument is returned when an instrument is invalid
	ErrInvalidInstrument = errors.New("invalid instrument")
)
