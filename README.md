# pools

The generic connection pool and task pool for Golang.

## Connection Pool

For more information pls refer to [examples/main.go](https://github.com/hedzr/pools/blob/master/examples/main.go):

```go
    pool := connpool.New(*poolSize,
        connpool.WithWorkerDialer(newWorkerWithOpts(WithClientKeepAliveTimeout(*keepAliveTimeout))),
        connpool.WithKeepAliveInterval(*keepAliveTimeout),
        connpool.WithBlockIfCantBorrow(true),
	)
	defer pool.Close()
    for i:=0; i<10; i++ {
        if c,ok:=pool.Borrow().(*clientSample); ok {
            c.LongOper()
        }
    }
```

### `connpool.Dialer`

The worker should implement `connpool.Dialer` interface for the pool intializing.

`connpool.Pool` will hold the connections.

### `connpool.KeepAliveTicker`

To keep the connection alive, your worker could implment `connpool.KeepAliveTicker` interface.

`connpool.Pool` will tick the workers periodically.

### `WithBlockIfCantBorrow(b)`

Generally the pool might return nil for `Borrow()` if all connections were borrowed.

But also `WithBlockIfCantBorrow(true)` may block at `Borrow()` till any connection returned by `Return()`.




## Task pool

TODO

## LICENSE

MIT
