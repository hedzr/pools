// Copyright © 2019 Hedzr Yeh.

package connpool_test

import (
	"github.com/hedzr/pools/connpool"
	"net"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	pool := connpool.New(10,
		connpool.WithKeepAliveInterval(12*time.Second),
		connpool.WithBlockIfCantBorrow(false),
		connpool.WithWorkerDialer(newWorker),
	)
	defer pool.Close()

	t.Logf("pool cap = %v", pool.Cap())
	t.Logf("pool borrowed = %v", pool.Borrowed())
	t.Logf("pool free = %v", pool.Free())
	pool.Resize(0)
	pool.Resize(10)
	pool.Resize(7)
	t.Logf("pool cap = %v, resizing to 13", pool.Cap())
	pool.Resize(13)
	t.Logf("pool cap = %v, resized", pool.Cap())

	var workers []connpool.Worker
	for i := 0; i < 5; i++ {
		w := pool.Borrow()
		workers = append(workers, w)
	}
	t.Logf("pool borrowed = %v", pool.Borrowed())
	for i := 0; i < 3; i++ {
		pool.Return(workers[i])
	}

	t.Logf("pool cap = %v", pool.Cap())
	t.Logf("pool borrowed = %v", pool.Borrowed())
	t.Logf("pool free = %v", pool.Free())
}

func newWorker() (w connpool.Worker, err error) {
	wz := &ww{}
	err = wz.open()
	w = wz
	return
}

type ww struct {
	conn net.Conn
}

func (w *ww) Close() (err error) {
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	return
}

func (w *ww) open() (err error) {
	w.conn, err = net.Dial("tcp", "163.com:80")
	return
}

func (w *ww) Dial(network, address string) (net.Conn, error) {
	panic("implement me")
}

func (w *ww) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	panic("implement me")
}
