// Copyright © 2019 Hedzr Yeh.

package jobs

import (
	"log"
	"sync"
	"sync/atomic"
)

type (
	// Scheduler is jobs scheduler,
	Scheduler interface {
		// Close releases all internal resources, resets the internal
		// states, waits for all jobs ended, and waits for all
		// workers terminated.
		Close() (err error)
		// CloseAndWait is a synonym of Close()
		CloseAndWait()

		// Cap returns the capacity of workers pool
		Cap() int
		// Resize(newSize int)

		Pause()
		Resume()

		// Schedule puts a job into a internal queue to wait for a worker ready to load it.
		Schedule(job JobIndexed, args ...interface{})
		// Schedule puts a job 'copies' copies into a internal queue to wait for a worker ready to load it.
		ScheduleN(job JobIndexed, copies int, args ...interface{})

		// WaitForAllJobs waits for all scheduled jobs done
		WaitForAllJobs()
	}

	// Worker is worker,
	xWorker interface {
		Close() error
		// Put(job Job, onEnd OnEndFunc, args ...interface{})
	}

	// SimpleJob is job
	SimpleJob interface {
		Run(args ...interface{})
	}

	// Job is job
	Job interface {
		Run(args ...interface{}) (res Result, err error)
	}

	// JobIndexed is job
	JobIndexed interface {
		Run(workerIndex, subIndex int, args ...interface{}) (res Result, err error)
	}

	// Result is result
	Result interface {
	}

	// OnEndFunc is a callback that would be invoked on each job ending
	OnEndFunc func(result Result, err error, job JobIndexed, args ...interface{})

	// Opt is for new scheduler entry
	Opt func(pool Scheduler)
)

// WrapSimpleJob wrap a SimpleJob object as a JobIndexed
func WrapSimpleJob(job SimpleJob) JobIndexed {
	return &sjobIndexed{job}
}

func WrapJob(job Job) JobIndexed {
	return &jobIndexed{job}
}

// WithOnEndCallback sets the OnEndFunc callback for jobs.Pool
func WithOnEndCallback(onEnd OnEndFunc) Opt {
	return func(pool Scheduler) {
		if p, ok := pool.(*poolZ); ok {
			p.onEnd = onEnd
		}
	}
}

// New return a job scheduler object instance.
// Closing it if you never need it any more
func New(initialSize int, opts ...Opt) (pool Scheduler) {
	p := &poolZ{
		capacity:    initialSize,
		maxCapacity: initialSize,
		done:        make(chan struct{}),
		closed:      make(chan struct{}),
		jobCh:       make(chan *jobTaskBlock, initialSize),
		// runningCond: sync.NewCond(&sync.Mutex{}),
		// doneCond:    sync.NewCond(&sync.Mutex{}),
	}

	for _, opt := range opts {
		opt(p)
	}

	for i := 0; i < initialSize; i++ {
		w := newWorker(i+1, &p.wgForWorkers, p.jobCh)
		p.workers = append(p.workers, w)
	}

	go p.run()

	pool = p
	return
}

type poolZ struct {
	capacity    int
	maxCapacity int

	// doneCond     *sync.Cond
	// rw           sync.RWMutex
	// runningCond  *sync.Cond
	// runningCount int32
	// running      map[xWorker]bool
	done         chan struct{}
	closed       chan struct{}
	exited       int32
	wgForWorkers sync.WaitGroup
	wgForJobs    sync.WaitGroup
	jobCh        chan *jobTaskBlock
	workers      []xWorker
	onEnd        OnEndFunc
}

func (p *poolZ) run() {
	for {
		select {
		case <-p.done:
			p.wgForWorkers.Wait() // waiting for all workers shutting down normally
			log.Printf("poolZ waiting done. CLOSED.")
			close(p.closed)
			return
		}
	}
}

func (p *poolZ) CloseAndWait() {
	_ = p.Close()
}

func (p *poolZ) Close() (err error) {
	if err = p.close(); err != nil {
		log.Printf("pool close failed: %v", err)
	}
	<-p.closed
	return
}

func (p *poolZ) close() (err error) {
	if atomic.LoadInt32(&p.exited) == 0 {
		atomic.AddInt32(&p.exited, 1)
		close(p.done)

		p.WaitForAllJobs()
		for _, w := range p.workers {
			if err := w.Close(); err != nil {
				log.Printf("close worker failed: %v", err)
			}
		}
		p.workers = nil
	}

	// if p.doneCond != nil {
	// 	p.doneCond.Broadcast()
	// }

	return
}

func (p *poolZ) Cap() int {
	return p.capacity
}

func (p *poolZ) Pause() {
}

func (p *poolZ) Resume() {
}

func (p *poolZ) Schedule(job JobIndexed, args ...interface{}) {
	if atomic.LoadInt32(&p.exited) == 0 {
		p.jobCh <- &jobTaskBlock{job: job, subIndex: 0, args: args, onEnd: p.doOnEnd}
		p.wgForJobs.Add(1)
	}
}

func (p *poolZ) ScheduleN(job JobIndexed, copies int, args ...interface{}) {
	if atomic.LoadInt32(&p.exited) == 0 {
		for i := 0; i < copies; i++ {
			p.jobCh <- &jobTaskBlock{job: job, subIndex: i, args: args, onEnd: p.doOnEnd}
		}
		p.wgForJobs.Add(copies)
	}
}

func (p *poolZ) doOnEnd(result Result, err error, jobSrc JobIndexed, argsSrc ...interface{}) {
	p.wgForJobs.Done()
	if p.onEnd != nil {
		p.onEnd(result, err, jobSrc, argsSrc...)
	}
	return
}

func (p *poolZ) WaitForAllJobs() {
	p.wgForJobs.Wait()
}
