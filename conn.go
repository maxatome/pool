package pool

import (
	"net/rpc"
)

type RpcAble interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
	Go(serviceMethod string, args interface{}, reply interface{}, done chan *rpc.Call) *rpc.Call
	Close() error
}

// PoolRconn is a wrapper around RpcAble to modify the behavior of
// RpcAble's Close() method.
type PoolRconn struct {
	RpcAble
	c        *channelPool
	unusable bool
}

// Close() puts the given rconn back to the pool instead of closing it.
func (p PoolRconn) Close() error {
	if p.unusable {
		if p.RpcAble != nil {
			return p.RpcAble.Close()
		}
		return nil
	}
	return p.c.put(p.RpcAble)
}

// MarkUnusable() marks the rconn not usable any more, to let the
// pool close it instead of returning it to pool.
func (p *PoolRconn) MarkUnusable() {
	p.unusable = true
}

// wrapRconn wraps a standard RpcAble to a PoolRconn RpcAble.
func (c *channelPool) wrapRconn(rconn RpcAble) RpcAble {
	return &PoolRconn{
		RpcAble: rconn,
		c:       c,
	}
}
