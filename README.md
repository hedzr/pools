# pools

[![Build Status](https://travis-ci.org/hedzr/pools.svg?branch=master)](https://travis-ci.org/hedzr/pools)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/hedzr/pools.svg?label=release)](https://github.com/hedzr/pools/releases)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/hedzr/pools) 
<!-- [![Go Report Card](https://goreportcard.com/badge/github.com/hedzr/pools)](https://goreportcard.com/report/github.com/hedzr/pools)
[![codecov](https://codecov.io/gh/hedzr/pools/branch/master/graph/badge.svg)](https://codecov.io/gh/hedzr/pools)
-->

The generic connection pool and task pool for Golang.

## Overview

- WIP, v1.1.0, the first final release
  - [conn-pool](#connection-pool)
  - [jobs *Scheduler*](#jobs-scheduler)
- v1.0.x preview versions
- v0.9.x pre-released





## Pools

### 1. Connection Pool

#### import

```go
import (
  "github.com/hedzr/pools/connpool"
)
```

For more information pls refer to [examples/connpooldemo/main.go](https://github.com/hedzr/pools/blob/master/examples/connpooldemo/main.go):

```go
    pool := connpool.New(*poolSize,
        connpool.WithWorkerDialer(newWorkerWithOpts(WithClientKeepAliveTimeout(*keepAliveTimeout))),
        connpool.WithKeepAliveInterval(*keepAliveTimeout),
        connpool.WithBlockIfCantBorrow(true),
	)
	defer pool.Close()

    for i := 0; i < 10; i++ {
        if c,ok:=pool.Borrow().(*clientSample); ok {
            c.LongOper()
        }
    }
```

#### `connpool.Dialer`

A `Dialer` function feed to pool make it can be initialized implicitly.

`connpool.Pool` will hold the connections.

As a sample:

```go
func newWorker() (w connpool.Worker, err error) {
	w, err = newWorkerWithOpts()()
	return
}

func newWorkerWithOpts(opts ...ClientSampleOpt) connpool.Dialer {
	return func() (w connpool.Worker, err error) {
		c := &clientSample{
			keepAliveTimeout: 15 * time.Second,
			sendCh:           make(chan string),
			doneCh:           make(chan struct{}),
		}
		err = c.open()
		if err == nil {
			for _, opt := range opts {
				opt(c)
			}
		}
		w = c
		return
	}
}
```


#### `connpool.KeepAliveTicker`

To keep the connection alive, your worker could implement `connpool.KeepAliveTicker` interface.

`connpool.Pool` will tick the workers periodically.

A good form is:

```go
func (c *clientSample) Tick(tick time.Time) (err error) {
	c.sendCh <- "PING"
	return
}
```


#### `WithBlockIfCantBorrow(b)`

Generally the pool might return nil for `Borrow()` if all connections in pool had been borrowed.

But also `WithBlockIfCantBorrow(true)` can block at `Borrow()` till any connection returned by `Return()`.




### 2. Jobs Scheduler


#### import

```go
import (
  "github.com/hedzr/pools/jobs"
)
```

For more information pls refer to [examples/jobsdemo/main.go](https://github.com/hedzr/pools/blob/master/examples/jobsdemo/main.go):

```go
func testEntry(){
	pool := jobs.New(30, jobs.WithOnEndCallback(func(result jobs.Result, err error, job jobs.Job, args ...interface{}) {
		return
	}))
	defer pool.CloseAndWait()

	for i := 0; i < 100; i++ {
		pool.Schedule(newJob(i), i+1, i+2, i+3)
		si := 1 + rand.Intn(10)
		pool.ScheduleN(newJob2(i, si), si, i+1, i+2, i+3)
	}

	// t.Logf("pool size: %v", pool.Cap())
	// pool.Pause()
	// pool.Resume()

	pool.WaitForIdle()
}


func newJob(i int) jobs.Job {
	return &job1{taskIndex: i}
}

func newJob2(i, si int) jobs.Job {
	return &job1{taskIndex: i, taskSubIndex: si}
}

type job1 struct {
	taskIndex    int
	taskSubIndex int
}

func (j *job1) Run(workerIndex int, args ...interface{}) (res jobs.Result, err error) {
	fmt.Printf("Task #%v [worker #%v]: args = %v\n", j.taskIndex, workerIndex, args)
	time.Sleep(time.Duration(2+rand.Intn(2)) * time.Second)
	return
}

```


### 3. Work-pool

Work-pool is a jobs scheduler but using a generator to feed the tasks.

For example:

```go

func testWorkPool() {
	pool := jobs.NewWorkPool(10)
	defer pool.Wait()
	
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

```

For more information pls refer to [examples/jobsdemo/main.go](https://github.com/hedzr/pools/blob/master/examples/jobsdemo/main.go).



## LICENSE

MIT
