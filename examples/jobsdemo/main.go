// Copyright © 2019 Hedzr Yeh.

package main

import (
	"fmt"
	"github.com/hedzr/pools/jobs"
	"math/rand"
	"time"
)

func main() {
	pool := jobs.New(30, jobs.WithOnEndCallback(func(result jobs.Result, err error, job jobs.JobIndexed, args ...interface{}) {
		return
	}))
	defer pool.CloseAndWait()

	testPool(pool)
	testSimpleJob(pool)
}

func testSimpleJob(pool jobs.Scheduler) {
	fmt.Println("\nsimple job testing -------------")
	for i := 0; i < 50; i++ {
		pool.Schedule(newSimpleJob(i), i+1, i+2, i+3)
		si := 1 + rand.Intn(10)
		pool.ScheduleN(newJob2(i, si), si, i+1, i+2, i+3)
	}

	fmt.Printf("pool size: %v\n", pool.Cap())
	pool.Pause()
	pool.Resume()

	pool.WaitForAllJobs()
}

func testPool(pool jobs.Scheduler) {
	fmt.Println("\nindexed job testing -------------")
	for i := 0; i < 100; i++ {
		pool.Schedule(newJob(i), i+1, i+2, i+3)
		si := 1 + rand.Intn(10)
		pool.ScheduleN(newJob2(i, si), si, i+1, i+2, i+3)
	}

	fmt.Printf("pool size: %v\n", pool.Cap())
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

func newSimpleJob(i int) jobs.JobIndexed {
	return jobs.WrapSimpleJob(&jobSimple{})
}

type jobSimple struct {
}

func (j *jobSimple) Run(args ...interface{}) {
	fmt.Printf("Task #?: args = %v\n", args)
	time.Sleep(time.Duration(1+rand.Intn(1)) * time.Second)
	return
}
