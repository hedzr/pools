// Copyright © 2019 Hedzr Yeh.

package jobs

import "sync"

type (
	Scheduler interface {
		Close() (err error)

		Cap() int
		// Resize(newSize int)

		Pause()
		Resume()

		Schedule(job Job, args ...interface{})
		ScheduleN(job Job, copies int, args ...interface{})
	}

	Job interface {
		Run(args ...interface{})
	}

	Worker func()

	Opt func(pool Scheduler)
)

func New(initialSize int, opts ...Opt) (pool Scheduler) {
	p := &poolZ{
		capacity:    initialSize,
		maxCapacity: initialSize,
		done:        make(chan struct{}),
		doneCond:    sync.NewCond(&sync.Mutex{}),
	}

	for _, opt := range opts {
		opt(p)
	}

	for i := 0; i < initialSize; i++ {
		newWorker(p.done)
	}

	go p.run()

	pool = p
	return
}

type poolZ struct {
	capacity    int
	maxCapacity int

	done     chan struct{}
	doneCond *sync.Cond
	rw       sync.RWMutex
	workers  []Worker
}

func (p *poolZ) run() {
	for {
		select {
		case <-p.done:
			return
		}
	}
}

func (p *poolZ) Close() (err error) {
	if p.done != nil {
		close(p.done)
		p.done = nil
	}
	if p.doneCond != nil {
		p.doneCond.Broadcast()
	}
	return
}

func (p *poolZ) Cap() int {
	return p.capacity
}

func (p *poolZ) Pause() {
	panic("implement me")
}

func (p *poolZ) Resume() {
	panic("implement me")
}

func (p *poolZ) Schedule(job Job, args ...interface{}) {
	panic("implement me")
}

func (p *poolZ) ScheduleN(job Job, copies int, args ...interface{}) {
	panic("implement me")
}

type workerZ struct {
	jobCh chan Job
}

func newWorker(done chan struct{}) *workerZ {
	w := &workerZ{jobCh: make(chan Job, 1)}
	go w.run(done)
	return w
}

func (w *workerZ) run(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		case job := <-w.jobCh:
			job.Run()
		}
	}
}
