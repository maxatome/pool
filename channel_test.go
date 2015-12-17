package pool

import (
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"testing"
	"time"
)

var (
	InitialCap = 5
	MaximumCap = 30
	network    = "tcp"
	address    = "127.0.0.1:7777"
	factory    = func() (RpcAble, error) {
		conn, err := net.Dial(network, address)
		if err != nil {
			return nil, err
		}
		return rpc.NewClient(conn), nil
	}
)

func init() {
	// used for factory function
	go simpleTCPServer()
	time.Sleep(time.Millisecond * 300) // wait until tcp server has been settled

	rand.Seed(time.Now().UTC().UnixNano())
}

func TestNew(t *testing.T) {
	_, err := newChannelPool()
	if err != nil {
		t.Errorf("New error: %s", err)
	}
}
func TestPool_Get_Impl(t *testing.T) {
	p, _ := newChannelPool()
	defer p.Close()

	rconn, err := p.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}

	_, ok := rconn.(*PoolRconn)
	if !ok {
		t.Errorf("Rconn is not of type PoolRconn")
	}
}

func TestPool_Get(t *testing.T) {
	p, _ := newChannelPool()
	defer p.Close()

	_, err := p.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}

	// after one get, current capacity should be lowered by one.
	if p.Len() != (InitialCap - 1) {
		t.Errorf("Get error. Expecting %d, got %d",
			(InitialCap - 1), p.Len())
	}

	// get them all
	var wg sync.WaitGroup
	for i := 0; i < (InitialCap - 1); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.Get()
			if err != nil {
				t.Errorf("Get error: %s", err)
			}
		}()
	}
	wg.Wait()

	if p.Len() != 0 {
		t.Errorf("Get error. Expecting %d, got %d",
			(InitialCap - 1), p.Len())
	}

	_, err = p.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}
}

func TestPool_Put(t *testing.T) {
	p, err := NewChannelPool(0, 30, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// get/create from the pool
	rconns := make([]RpcAble, MaximumCap)
	for i := 0; i < MaximumCap; i++ {
		rconn, _ := p.Get()
		rconns[i] = rconn
	}

	// now put them all back
	for _, rconn := range rconns {
		rconn.Close()
	}

	if p.Len() != MaximumCap {
		t.Errorf("Put error len. Expecting %d, got %d",
			1, p.Len())
	}

	rconn, _ := p.Get()
	p.Close() // close pool

	rconn.Close() // try to put into a full pool
	if p.Len() != 0 {
		t.Errorf("Put error. Closed pool shouldn't allow to put rconns.")
	}
}

func TestPool_PutUnusableRconn(t *testing.T) {
	p, _ := newChannelPool()
	defer p.Close()

	// ensure pool is not empty
	rconn, _ := p.Get()
	rconn.Close()

	poolSize := p.Len()
	rconn, _ = p.Get()
	rconn.Close()
	if p.Len() != poolSize {
		t.Errorf("Pool size is expected to be equal to initial size")
	}

	rconn, _ = p.Get()
	if pc, ok := rconn.(*PoolRconn); !ok {
		t.Errorf("impossible")
	} else {
		pc.MarkUnusable()
	}
	rconn.Close()
	if p.Len() != poolSize-1 {
		t.Errorf("Pool size is expected to be initial_size - 1", p.Len(), poolSize-1)
	}
}

func TestPool_UsedCapacity(t *testing.T) {
	p, _ := newChannelPool()
	defer p.Close()

	if p.Len() != InitialCap {
		t.Errorf("InitialCap error. Expecting %d, got %d",
			InitialCap, p.Len())
	}
}

func TestPool_Close(t *testing.T) {
	p, _ := newChannelPool()

	// now close it and test all cases we are expecting.
	p.Close()

	c := p.(*channelPool)

	if c.rconns != nil {
		t.Errorf("Close error, rconns channel should be nil")
	}

	if c.factory != nil {
		t.Errorf("Close error, factory should be nil")
	}

	_, err := p.Get()
	if err == nil {
		t.Errorf("Close error, get rconn should return an error")
	}

	if p.Len() != 0 {
		t.Errorf("Close error used capacity. Expecting 0, got %d", p.Len())
	}
}

func TestPoolConcurrent(t *testing.T) {
	p, _ := newChannelPool()
	pipe := make(chan RpcAble, 0)

	go func() {
		p.Close()
	}()

	for i := 0; i < MaximumCap; i++ {
		go func() {
			rconn, _ := p.Get()

			pipe <- rconn
		}()

		go func() {
			rconn := <-pipe
			if rconn == nil {
				return
			}
			rconn.Close()
		}()
	}
}

//func TestPoolWriteRead(t *testing.T) {
//	p, _ := NewChannelPool(0, 30, factory)
//
//	rconn, _ := p.Get()
//
//	conn := rconn.(net.Conn)
//
//	msg := "hello"
//	_, err := conn.Write([]byte(msg))
//	if err != nil {
//		t.Error(err)
//	}
//}

func TestPoolConcurrent2(t *testing.T) {
	p, _ := NewChannelPool(0, 30, factory)

	var wg sync.WaitGroup

	go func() {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				rconn, _ := p.Get()
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				rconn.Close()
				wg.Done()
			}(i)
		}
	}()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			rconn, _ := p.Get()
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			rconn.Close()
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func newChannelPool() (Pool, error) {
	return NewChannelPool(InitialCap, MaximumCap, factory)
}

func simpleTCPServer() {
	l, err := net.Listen(network, address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			buffer := make([]byte, 256)
			conn.Read(buffer)
		}()
	}
}
