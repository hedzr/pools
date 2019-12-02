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

		Cap() int32
		Resize(newSize int32)

		Borrow() Worker
		Return(Worker)
		Borrowed() (count int32)
		Free() (count int32)
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
	// todo deprecate sync.Map and use locker and map to synchronize both the map and capacity fields.
	// locker sync.Mutex
	// workers map[Worker]bool
	dialer            Dialer
	workers           sync.Map // key: Worker, val: bool
	capacity          int32
	done              chan struct{}
	exited            int32
	keepAliveInterval time.Duration
	blockIfCantBorrow bool
}

func (p *poolZ) Close() (err error) {
	if atomic.LoadInt32(&p.exited) == 0 {
		atomic.AddInt32(&p.exited, 1)
		close(p.done)
	}

	p.workers.Range(func(key, value interface{}) bool {
		if c, ok := key.(io.Closer); ok {
			if err = c.Close(); err != nil {
				log.Printf("close worker (%v) failed: %v", c, err)
			}
		}
		p.workers.Delete(key)
		atomic.AddInt32(&p.capacity, -1)
		return true
	})

	return
}

func (p *poolZ) Cap() (count int32) {
	count = atomic.LoadInt32(&p.capacity)
	return
}

func (p *poolZ) Resize(newSize int32) {
	count := atomic.LoadInt32(&p.capacity)
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
			atomic.AddInt32(&p.capacity, 1)
		}
	}
}

func (p *poolZ) Borrowed() (count int32) {
	p.workers.Range(func(key, value interface{}) bool {
		if used, ok := value.(bool); ok && used {
			count++
		}
		return true
	})
	// count = atomic.LoadInt32(&p.capacity)
	return
}

func (p *poolZ) Free() (count int32) {
	count = atomic.LoadInt32(&p.capacity)
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
						atomic.AddInt32(&p.capacity, -1)
						go func() {
							if w, err := p.dialer(); err == nil {
								p.workers.Store(w, false)
								atomic.AddInt32(&p.capacity, 1)
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
