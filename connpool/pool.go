package connpool

import (
	"log"
	"sync"
	"time"
)

type (
	Pool interface {
		Close()
		Cap() int
		Resize(newSize int)

		Borrow() Worker
		Return(Worker)
		Borrowed() (count int)
		Free() (count int)
	}

	Worker interface {
	}

	Closable interface {
		Close() error
	}

	KeepAliveTicker interface {
		Tick(tick time.Time) (err error)
	}

	PoolOpt func(*poolZ)
)

type poolZ struct {
	// locker sync.Mutex
	// workers map[Worker]bool
	dialer            func() Worker
	workers           sync.Map // key: Worker, val: bool
	size              int
	done              chan struct{}
	keepAliveInterval time.Duration
	blockIfCantBorrow bool
}

func (p *poolZ) Close() {
	if p.done != nil {
		close(p.done)
		p.done = nil
	}

	p.workers.Range(func(key, value interface{}) bool {
		if c, ok := key.(Closable); ok {
			if err := c.Close(); err != nil {
				log.Printf("close worker (%v) failed: %v", c, err)
			}
		}
		p.workers.Delete(key)
		p.size--
		return true
	})
}

func (p *poolZ) Cap() int {
	return p.size
}

func (p *poolZ) Resize(newSize int) {
	if newSize == p.size {
		return
	}
	if newSize <= 0 {
		return
	}
	if newSize < p.size {
		return
	}

	for i := p.size; i < newSize; i++ {
		p.workers.Store(p.dialer(), false)
	}
	p.size = newSize
}

func (p *poolZ) Borrowed() (count int) {
	p.workers.Range(func(key, value interface{}) bool {
		if used, ok := value.(bool); ok && !used {
			count++
		}
		return true
	})
	return
}

func (p *poolZ) Free() (count int) {
	return p.size - p.Borrowed()
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
		time.Sleep(30 * time.Nanosecond)
		goto RetryBorrow
	}
	return
}

func (p *poolZ) Return(t Worker) {
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
						p.size--
						go func() {
							p.workers.Store(p.dialer(), false)
							p.size++
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

func WithWorkerDialer(dialer func() Worker) PoolOpt {
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
			pool.workers.Store(pool.dialer(), false)
		}
	}

	go pool.run()

	return pool
}
