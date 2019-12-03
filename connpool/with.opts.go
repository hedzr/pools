// Copyright © 2019 Hedzr Yeh.

package connpool

import "time"

// New new a connection-pool object for you.
// Note that always close this returned pool once you don't need it.
//
//    pool := connpool.New(10)
//    defer pool.Close()
//
func New(size int, opts ...PoolOpt) Pool {
	pool := &poolZ{
		capacity: int32(size),
		done:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(pool)
	}

	if pool.keepAliveInterval == 0 {
		pool.keepAliveInterval = 120 * time.Second
	}

	for i := 0; i < size; i++ {
		if pool.dialer != nil {
			if w, err := pool.dialer(); err == nil {
				pool.workers.Store(w, false)
			}
		}
	}

	go pool.run()

	return pool
}

// WithWorkerDialer declares a dialer for creating each connection worker in pool.
// The pool will dial up to the remote resource (such as tcp/http/mqtt server...)
// via the dialer.
// Your dialer function should return a 'Worker' object with error annotation.
func WithWorkerDialer(dialer Dialer) PoolOpt {
	return func(z *poolZ) {
		z.dialer = dialer
	}
}

// WithKeepAliveInterval declares a time interval to tell the pool triggering
// all workers 'Tick()' to server periodically.
func WithKeepAliveInterval(d time.Duration) PoolOpt {
	return func(z *poolZ) {
		z.keepAliveInterval = d
	}
}

// WithBlockIfCantBorrow with true value the pool.Borrow() will block itself
// until any worker is idle to usable.
func WithBlockIfCantBorrow(b bool) PoolOpt {
	return func(z *poolZ) {
		z.blockIfCantBorrow = b
	}
}
