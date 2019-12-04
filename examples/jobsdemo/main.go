// Copyright © 2019 Hedzr Yeh.

package main

import (
	"fmt"
	"github.com/hedzr/pools/jobs"
	"math/rand"
	"sync/atomic"
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
	start := time.Now()
	numTasks := 50
	defer func() {
		pool.WaitForAllJobs()
		fmt.Printf("Took %fs to ship %d jobs with %v times.\n", time.Since(start).Seconds(), numTasks, simpleJobCounter)
	}()

	for i := 0; i < numTasks; i++ {
		pool.Schedule(newSimpleJob(i), i+1, i+2, i+3)
		// si := 1 + rand.Intn(10)
		// pool.ScheduleN(newSimpleJob2(i, si), si, i+1, i+2, i+3)
	}

	fmt.Printf("pool size: %v\n", pool.Cap())
	pool.Pause()
	pool.Resume()
}

func testPool(pool jobs.Scheduler) {
	fmt.Println("\nindexed job testing -------------")
	start := time.Now()
	numTasks := 100
	defer func() {
		pool.WaitForAllJobs()
		fmt.Printf("Took %fs to ship %d jobs with %v times.\n", time.Since(start).Seconds(), numTasks, jobCounter)
	}()

	for i := 0; i < numTasks; i++ {
		pool.Schedule(newJob(i), i+1, i+2, i+3)
		// si := 1 + rand.Intn(10)
		// pool.ScheduleN(newJob2(i, si), si, i+1, i+2, i+3)
	}

	fmt.Printf("pool size: %v\n", pool.Cap())
	pool.Pause()
	pool.Resume()

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
	start := time.Now()
	atomic.AddInt32(&jobCounter, 1)
	time.Sleep(time.Duration(100+rand.Intn(1500)) * time.Millisecond)
	fmt.Printf("Task #%v [worker #%v]: args = %v, time = %v ms\n", j.taskIndex, workerIndex, args, time.Now().Sub(start).Milliseconds())
	return
}

func newSimpleJob(i int) jobs.JobIndexed {
	return jobs.WrapSimpleJob(&jobSimple{})
}

type jobSimple struct {
}

func (j *jobSimple) Run(args ...interface{}) {
	start := time.Now()
	atomic.AddInt32(&simpleJobCounter, 1)
	time.Sleep(time.Duration(100+rand.Intn(3700)) * time.Millisecond)
	fmt.Printf("Task #?: args = %v, time = %v ms\n", args, time.Now().Sub(start).Milliseconds())
	return
}

var (
	simpleJobCounter int32
	jobCounter       int32
)
