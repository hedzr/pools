// Copyright © 2019 Hedzr Yeh.

package jobs

import (
	"sync"
)

type (
	WorkPool interface {
		//
	}
	WorkPoolOpt func(p WorkPool)
)

func NewWorkPool(size int, opts ...WorkPoolOpt) *workPool {
	return newWorkPool(size, opts...)
}

type workPool struct {
	capacity     int
	done         chan struct{}
	wg           sync.WaitGroup
	multiplex    []func(c <-chan *Task)
	outgoingJobs chan *Task
	onComplete   func(numProcessed int)
	workers      []<-chan *Task
}

func newWorkPool(size int, opts ...WorkPoolOpt) *workPool {
	p := &workPool{
		capacity: size,
		done:     make(chan struct{}),
		// channels:  make([]chan *Task, size),
		// multiplex: make([]func(c <-chan *Task), size),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *workPool) OnComplete(onComplete func(numProcessed int)) *workPool {
	p.onComplete = onComplete
	return p
}

func (p *workPool) Run(generator func(args ...interface{}) chan *Task, args ...interface{}) {
	items := p.getDefaultPrepareFunctor(generator, args...)(p.done)

	p.workers = make([]<-chan *Task, p.capacity)
	for i := 0; i < p.capacity; i++ {
		p.workers[i] = p.getProcessorFunctor(p.done, items, i)
	}

	var numProcessed int
	for range p.merge(p.done, p.workers...) {
		numProcessed++
	}

	if p.onComplete != nil {
		p.onComplete(numProcessed)
	}
}

func (p *workPool) merge(done <-chan struct{}, channels ...<-chan *Task) <-chan *Task {
	p.outgoingJobs = make(chan *Task)
	// p.channels = make([]chan *Task, len(channels))
	p.wg.Add(len(channels))
	p.multiplex = make([]func(c <-chan *Task), len(channels))
	for i, ch := range channels {
		// p.channels[i] = make(chan *Task)
		p.multiplex[i] = func(c <-chan *Task) {
			defer p.wg.Done()
			for i := range c {
				select {
				case <-p.done:
					return
				case p.outgoingJobs <- i:
				}
			}
		}
		go p.multiplex[i](ch)
	}
	go func() {
		p.wg.Wait()
		if p.outgoingJobs != nil {
			close(p.outgoingJobs)
		}
	}()
	return p.outgoingJobs
}

func (p *workPool) getDefaultPrepareFunctor(generator func(args ...interface{}) chan *Task, args ...interface{}) func(done <-chan struct{}) <-chan *Task {
	return func(done <-chan struct{}) <-chan *Task {
		items := make(chan *Task)
		go func() {
			// var itemsToShip []*jobs.Task
			for item := range generator(args...) {
				select {
				case <-done:
					return
				case items <- item:
					// fmt.Printf("%5d. job extracted\n", item.Job.(*job1).taskIndex)
				}
			}
			close(items)
		}()
		return items
	}
}
func (p *workPool) getProcessorFunctor(done <-chan struct{}, items <-chan *Task, workerId int) <-chan *Task {
	packages := make(chan *Task)
	go func() {
		for item := range items {
			select {
			case <-done:
				return
			case packages <- item:
				item.result, item.err = item.Job.Run(workerId, item.subIndex, item.args...)
				if item.onEnd != nil {
					item.onEnd(item.result, item.err, item.Job, item.args...)
				}
			}
		}
		close(packages)
	}()
	return packages
}

// func (p *workPool) Process(onProcess func(done <-chan struct{}, items <-chan *Task, workerId int) <-chan Result) {
// 	p.onProcess = onProcess
// }

func (p *workPool) Wait() {

}

// Task struct for NewWorkPool
type Task struct {
	Job      JobIndexed
	subIndex int
	args     []interface{}
	onEnd    OnEndFunc
	result   Result
	err      error
}

// ToTask wrap Task with JobIndexed struct and args
func ToTask(job JobIndexed, args ...interface{}) *Task {
	return &Task{
		Job:      job,
		subIndex: 0,
		args:     args,
		onEnd:    nil,
		result:   nil,
		err:      nil,
	}
}

// ToTaskN wrap Task with JobIndexed struct and args, and OnEndFunc
func ToTaskN(onEnd OnEndFunc, job JobIndexed, args ...interface{}) *Task {
	return &Task{
		Job:      job,
		subIndex: 0,
		args:     args,
		onEnd:    onEnd,
		result:   nil,
		err:      nil,
	}
}

func merge(done <-chan struct{}, channels ...<-chan *Task) <-chan *Task {
	var wg sync.WaitGroup

	wg.Add(len(channels))
	outgoingJobs := make(chan *Task)
	multiplex := func(c <-chan *Task) {
		defer wg.Done()
		for i := range c {
			select {
			case <-done:
				return
			case outgoingJobs <- i:
			}
		}
	}
	for _, c := range channels {
		go multiplex(c)
	}
	go func() {
		wg.Wait()
		close(outgoingJobs)
	}()
	return outgoingJobs
}

func prepare(done <-chan struct{}) <-chan *Task {
	items := make(chan *Task)
	itemsToShip := []*Task{
		// {0, "Shirt", 1 * time.Second},
		// {1, "Legos", 1 * time.Second},
		// {2, "TV", 5 * time.Second},
		// {3, "Bananas", 2 * time.Second},
		// {4, "Hat", 1 * time.Second},
		// {5, "Phone", 2 * time.Second},
		// {6, "Plates", 3 * time.Second},
		// {7, "Computer", 5 * time.Second},
		// {8, "Pint Glass", 3 * time.Second},
		// {9, "Watch", 2 * time.Second},
	}
	go func() {
		for _, item := range itemsToShip {
			select {
			case <-done:
				return
			case items <- item:
			}
		}
		close(items)
	}()
	return items
}
