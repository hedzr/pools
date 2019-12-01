# pools

The generic connection pool and task pool for Golang.

## Connection Pool

### import

```go
import (
  "github.com/hedzr/pools/connpool"
)
```

For more information pls refer to [examples/connpooldemo/main.go](https://github.com/hedzr/pools/blob/master/examples/connpooldemo/main.go):

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

A `Dialer` function feed to pool make it can be initialized implicitly.

`connpool.Pool` will hold the connections.

As a sample:

```go
func newWorker() (w connpool.Worker, err error) {
	w, err = newWorkerWithOpts()()
	return
}

func newWorkerWithOpts(opts ...ClientSampleOpt) connpool.Dialer {
	return func() (w connpool.Worker, err error) {
		c := &clientSample{
			keepAliveTimeout: 15 * time.Second,
			sendCh:           make(chan string),
			doneCh:           make(chan struct{}),
		}
		err = c.open()
		if err == nil {
			for _, opt := range opts {
				opt(c)
			}
		}
		w = c
		return
	}
}
```


### `connpool.KeepAliveTicker`

To keep the connection alive, your worker could implement `connpool.KeepAliveTicker` interface.

`connpool.Pool` will tick the workers periodically.

A good form is:

```go
func (c *clientSample) Tick(tick time.Time) (err error) {
	c.sendCh <- "PING"
	return
}
```


### `WithBlockIfCantBorrow(b)`

Generally the pool might return nil for `Borrow()` if all connections were borrowed.

But also `WithBlockIfCantBorrow(true)` can block at `Borrow()` till any connection returned by `Return()`.




## Task pool

TODO

## LICENSE

MIT
