// Copyright © 2019 Hedzr Yeh.

package jobs_test

import (
	"fmt"
	"github.com/hedzr/pools/jobs"
	"math/rand"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	pool := jobs.New(30, jobs.WithOnEndCallback(func(result jobs.Result, err error, job jobs.JobIndexed, args ...interface{}) {
		return
	}))
	defer pool.CloseAndWait()

	for i := 0; i < 100; i++ {
		pool.Schedule(newJob(i), i+1, i+2, i+3)
		si := 1 + rand.Intn(10)
		pool.ScheduleN(newJob2(i, si), si, i+1, i+2, i+3)
	}

	t.Logf("pool size: %v", pool.Cap())
	pool.Pause()
	pool.Resume()

	pool.WaitForAllJobs()
}

func newJob(i int) jobs.JobIndexed {
	return &job1{taskIndex: i}
}

func newJob2(i, si int) jobs.JobIndexed {
	return &job1{taskIndex: i, taskSubIndex: si}
}

type job1 struct {
	taskIndex    int
	taskSubIndex int
}

func (j *job1) Run(workerIndex, subIndex int, args ...interface{}) (res jobs.Result, err error) {
	fmt.Printf("Task #%v [worker #%v]: args = %v\n", j.taskIndex, workerIndex, args)
	time.Sleep(time.Duration(2+rand.Intn(2)) * time.Second)
	return
}
