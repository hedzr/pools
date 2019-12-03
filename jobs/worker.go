// Copyright © 2019 Hedzr Yeh.

package jobs

import (
	"github.com/hedzr/pools/exterr"
	"log"
	"sync"
	"sync/atomic"
)

type workerZ struct {
	// wg *sync.WaitGroup
	// jobCh chan *jobTaskBlock
	workerIndex int
	done        chan struct{}
	exited      int32
}

type jobTaskBlock struct {
	job      JobIndexed
	subIndex int
	args     []interface{}
	onEnd    OnEndFunc
}

func newWorker(i int, wg *sync.WaitGroup, jobCh chan *jobTaskBlock) *workerZ {
	w := &workerZ{
		workerIndex: i,
		done:        make(chan struct{}),
	}

	wg.Add(1)

	go w.run(w.done, wg, jobCh)

	return w
}

func (w *workerZ) Close() (err error) {
	if atomic.LoadInt32(&w.exited) == 0 {
		atomic.AddInt32(&w.exited, 1)
		close(w.done)
	}
	return
}

func (w *workerZ) run(done chan struct{}, wg *sync.WaitGroup, jobCh chan *jobTaskBlock) {
	for {
		select {
		case <-done:
			wg.Done()
			return
		case jtb := <-jobCh:
			w.runJob(jtb)
		}
	}
}

func (w *workerZ) runJob(jtb *jobTaskBlock) {
	var err error
	var res Result

	defer func() {
		if e := recover(); e != nil {
			log.Println(e)
			if e1, ok := e.(error); ok {
				err = exterr.NewError(e1, err)
			}
		}
		jtb.onEnd(res, err, jtb.job, jtb.args...)
	}()

	// jtb.workerIndex = w.workerIndex
	res, err = jtb.job.Run(w.workerIndex, jtb.subIndex, jtb.args...)
}
