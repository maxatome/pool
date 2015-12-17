# Pool [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/maxatome/pool)


Pool is a thread safe connection pool for pool.RpcAble interface
heavily based on https://godoc.org/gopkg.in/fatih/pool.v2

It can be used to manage and reuse connections used by net.rpc for
example.


## Install and Usage

Install the package with:

```bash
go get github.com/maxatome/pool
```

Import it with:

```go
import "github.com/maxatome/pool"
```

and use `pool` as the package name inside the code.

## Example

```go
// create a factory() to be used with channel based pool
factory := func() (pool.RpcAble, error) {
    conn, err := net.Dial(network, address)
    if err != nil {
        return nil, err
    }
    return rpc.NewClient(conn), nil
}

// create a new channel based pool with an initial capacity of 5 and maximum
// capacity of 30. The factory will create 5 initial RPC connections and put
// them into the pool.
p, err := pool.NewChannelPool(5, 30, factory)

// now you can get a RpcAble connection from the pool, if there is no connection
// available it will create a new one via the factory function.
rconn, err := p.Get()

// do something with rconn...
args := &Args{7, 8}
mulReply := new(Reply)
mulCall := rconn.Go("Arith.Mul", args, mulReply, nil)

// ...and put it back to the pool by closing it

// (this doesn't close the underlying RPC connection instead
// it's putting it back to the pool).
rconn.Close()

// close pool any time you want, this closes all the
// RPC connections inside a pool
p.Close()

// currently available RPC connections in the pool
current := p.Len()
```


## Credits

 * [Fatih Arslan](https://github.com/fatih)
 * [sougou](https://github.com/sougou)

## License

The MIT License (MIT) - see LICENSE for more details
