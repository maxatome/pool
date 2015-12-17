package pool

import (
	"errors"
	"fmt"
	"sync"
)

// channelPool implements the Pool interface based on buffered channels.
type channelPool struct {
	// storage for our RPC-able connections
	mu     sync.Mutex
	rconns chan RpcAble

	// RpcAble generator
	factory Factory
}

// Factory is a function to create new RPC-able connections.
type Factory func() (RpcAble, error)

// NewChannelPool returns a new pool based on buffered channels with
// an initial capacity and maximum capacity. Factory is used when
// initial capacity is greater than zero to fill the pool. A zero
// initialCap doesn't fill the Pool until a new Get() is
// called. During a Get(), If there is no new RPC-able connection
// available in the pool, a new RPC-able connection will be created
// via the Factory() method.
func NewChannelPool(initialCap, maxCap int, factory Factory) (Pool, error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	c := &channelPool{
		rconns:  make(chan RpcAble, maxCap),
		factory: factory,
	}

	// create initial RPC-able connections, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		rconn, err := factory()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.rconns <- rconn
	}

	return c, nil
}

func (c *channelPool) getRconns() chan RpcAble {
	c.mu.Lock()
	rconns := c.rconns
	c.mu.Unlock()
	return rconns
}

// Get implements the Pool interfaces Get() method. If there is no new
// RPC-able connection available in the pool, a new RPC-able
// connection will be created via the Factory() method.
func (c *channelPool) Get() (RpcAble, error) {
	rconns := c.getRconns()
	if rconns == nil {
		return nil, ErrClosed
	}

	// wrap our rconns with out custom RpcAble implementation (wrapRconn
	// method) that puts the RPC-able connection back to the pool if it's closed.
	select {
	case rconn := <-rconns:
		if rconn == nil {
			return nil, ErrClosed
		}

		return c.wrapRconn(rconn), nil
	default:
		rconn, err := c.factory()
		if err != nil {
			return nil, err
		}

		return c.wrapRconn(rconn), nil
	}
}

// put puts the rconn back to the pool. If the pool is full or closed,
// rconn is simply closed. A nil rconn will be rejected.
func (c *channelPool) put(rconn RpcAble) error {
	if rconn == nil {
		return errors.New("rconn is nil. rejecting")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.rconns == nil {
		// pool is closed, close passed rconn
		return rconn.Close()
	}

	// put the resource back into the pool. If the pool is full, this will
	// block and the default case will be executed.
	select {
	case c.rconns <- rconn:
		return nil
	default:
		// pool is full, close passed rconn
		return rconn.Close()
	}
}

func (c *channelPool) Close() {
	c.mu.Lock()
	rconns := c.rconns
	c.rconns = nil
	c.factory = nil
	c.mu.Unlock()

	if rconns == nil {
		return
	}

	close(rconns)
	for rconn := range rconns {
		rconn.Close()
	}
}

func (c *channelPool) Len() int { return len(c.getRconns()) }
