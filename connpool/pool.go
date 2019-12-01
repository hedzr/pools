package connpool

import (
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type (
	// Pool is a conn-pool
	Pool interface {
		Close() error

		Cap() int
		Resize(newSize int)

		Borrow() Worker
		Return(Worker)
		Borrowed() (count int)
		Free() (count int)
	}

	// Worker should be always have a io.Closer implementation if it wanna be disposed.
	Worker interface {
		// io.Closer
		// Dialer
	}

	// Dialer for a worker
	Dialer func() (w Worker, err error)

	// KeepAliveTicker will be ticked periodically by internal pool manager
	KeepAliveTicker interface {
		Tick(tick time.Time) (err error)
	}

	// PoolOpt for the functional options pattern
	PoolOpt func(*poolZ)
)

type poolZ struct {
	// locker sync.Mutex
	// workers map[Worker]bool
	dialer            Dialer
	workers           sync.Map // key: Worker, val: bool
	sizeA             int32
	done              chan struct{}
	keepAliveInterval time.Duration
	blockIfCantBorrow bool
}

func (p *poolZ) Close() (err error) {
	if p.done != nil {
		close(p.done)
		p.done = nil
	}

	p.workers.Range(func(key, value interface{}) bool {
		if c, ok := key.(io.Closer); ok {
			if err = c.Close(); err != nil {
				log.Printf("close worker (%v) failed: %v", c, err)
			}
		}
		p.workers.Delete(key)
		atomic.AddInt32(&p.sizeA, -1)
		return true
	})
	return
}

func (p *poolZ) Cap() (count int32) {
	count = atomic.LoadInt32(&p.sizeA)
	return
}

func (p *poolZ) Resize(newSize int32) {
	count := atomic.LoadInt32(&p.sizeA)
	if newSize == count {
		return
	}
	if newSize <= 0 {
		return
	}
	if newSize < count {
		return // todo decrease the pool
	}

	for i := count; i < newSize; i++ {
		if w, err := p.dialer(); err == nil {
			p.workers.Store(w, false)
			atomic.AddInt32(&p.sizeA, 1)
		}
	}
}

func (p *poolZ) Borrowed() (count int32) {
	// p.workers.Range(func(key, value interface{}) bool {
	// 	if used, ok := value.(bool); ok && used {
	// 		count++
	// 	}
	// 	return true
	// })
	count = atomic.LoadInt32(&p.sizeA)
	return
}

func (p *poolZ) Free() (count int32) {
	count = atomic.LoadInt32(&p.sizeA)
	count -= p.Borrowed()
	return
}

func (p *poolZ) Borrow() (ret Worker) {
RetryBorrow:
	p.workers.Range(func(key, value interface{}) bool {
		if used, ok := value.(bool); ok && !used {
			p.workers.Store(key, true)
			ret = key.(Worker)
			return false
		}
		return true
	})

	if p.blockIfCantBorrow && ret == nil {
		// time.Sleep(31 * time.Nanosecond)
		time.Sleep(37 * time.Millisecond)
		goto RetryBorrow
	}
	return
}

func (p *poolZ) Return(t Worker) {
	// if c, ok := t.(io.Closer); ok {
	// 	if err := c.Close(); err != nil {
	// 		log.Printf("close the closable worker failed: %v", err)
	// 	}
	// }
	p.workers.Store(t, false)
}

func (p *poolZ) run() {
	ticker := time.NewTicker(p.keepAliveInterval)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case tick := <-ticker.C:
			p.workers.Range(func(key, value interface{}) bool {
				if w, ok := key.(KeepAliveTicker); ok {
					if err := w.Tick(tick); err != nil {
						log.Printf("keep-alive tick on worker (%v) failed: %v", w, err)
						p.workers.Delete(w)
						atomic.AddInt32(&p.sizeA, -1)
						go func() {
							if w, err := p.dialer(); err == nil {
								p.workers.Store(w, false)
								atomic.AddInt32(&p.sizeA, 1)
							}
						}()
					}
				}
				return true
			})
		case <-p.done:
			return
		}
	}
}

func WithWorkerDialer(dialer Dialer) PoolOpt {
	return func(z *poolZ) {
		z.dialer = dialer
	}
}

func WithKeepAliveInterval(d time.Duration) PoolOpt {
	return func(z *poolZ) {
		z.keepAliveInterval = d
	}
}

func WithBlockIfCantBorrow(b bool) PoolOpt {
	return func(z *poolZ) {
		z.blockIfCantBorrow = b
	}
}

func New(size int, opts ...PoolOpt) Pool {
	pool := &poolZ{
		size: size,
		done: make(chan struct{}),
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
