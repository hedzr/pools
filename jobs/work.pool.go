// Copyright © 2019 Hedzr Yeh.

package jobs

import (
	"sync"
)

type (
	// WorkPool is a work-pool with channel fan-out pattern
	WorkPool interface {
		OnComplete(onComplete func(numProcessed int)) WorkPool
		Run(generator func(args ...interface{}) chan *Task, args ...interface{})
	}

	// WorkPoolOpt for WorkPool
	WorkPoolOpt func(p WorkPool)
)

// NewWorkPool new a work-pool object
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

// OnComplete will be invoked after the end of all tasks
func (p *workPool) OnComplete(onComplete func(numProcessed int)) WorkPool {
	p.onComplete = onComplete
	return p
}

// Run starts a group of tasks with optional arguments.
// 'generator' will supply all the new tasks via a channel of *Task.
//
// A sample generator:
//
//     generator := func(args ...interface{}) chan *jobs.Task {
//     	ch := make(chan *jobs.Task)
//     	count := args[0].(int)
//     	go func() {
//     		for i := 0; i < count; i++ {
//     			job := newJob(i)
//     			fmt.Printf("   -> new job #%d put\n", i)
//     			ch <- jobs.ToTask(job, i+1, i+2, i+3)
//     		}
//     		close(ch)
//     	}()
//     	return ch
//     }
//
// For a hub object, you can return the hub's sending channel via a functional generator too.
//
//    type hub struct {
//      sendCh chan *myTask
//    }
//    //
//    // hub's runlooper...
//    //
//
//    generator := func(args ...interface{}) chan *jobs.Task {
//        h:=&hub{...}
//        return h.sendCh
//    }
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
