// Copyright © 2019 Hedzr Yeh.

package jobs_test

import (
	"fmt"
	"github.com/hedzr/pools/jobs"
	"testing"
)

func TestWorkPool(t *testing.T) {
	pool := jobs.NewWorkPool(10)

	generator := func(args ...interface{}) chan *jobs.Task {
		ch := make(chan *jobs.Task)
		count := args[0].(int)
		go func() {
			for i := 0; i < count; i++ {
				job := newJob(i)
				fmt.Printf("   -> new job #%d put\n", i)
				ch <- jobs.ToTask(job, i+1, i+2, i+3)
			}
			close(ch)
		}()
		return ch
	}

	pool.OnComplete(func(numProcessed int) {
		fmt.Printf("processed %d tasks\n", numProcessed)
	}).Run(generator, 30)
}

func TestTaskN(t *testing.T) {
	jobs.ToTaskN(func(result jobs.Result, err error, job jobs.JobIndexed, args ...interface{}) {
		return
	}, nil, "today")
}
