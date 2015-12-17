// Package pool implements a pool of RpcAble interfaces to manage and reuse them.
package pool

import (
	"errors"
)

var (
	// ErrClosed is the error resulting if the pool is closed via pool.Close().
	ErrClosed = errors.New("pool is closed")
)

// Pool interface describes a pool implementation. A pool should have maximum
// capacity. An ideal pool is threadsafe and easy to use.
type Pool interface {
	// Get returns a new RPC-able connection from the pool. Closing a
	// RPC-able connection puts it back to the Pool. Closing it when the
	// pool is destroyed or full will be counted as an error.
	Get() (RpcAble, error)

	// Close closes the pool and all its RPC-able connections. After
	// Close() the pool is no longer usable.
	Close()

	// Len returns the current number of RPC-able connections of the pool.
	Len() int
}
