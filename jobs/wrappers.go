// Copyright © 2019 Hedzr Yeh.

package jobs

type sjobIndexed struct {
	job SimpleJob
}

func (j *sjobIndexed) Run(workerIndex, subIndex int, args ...interface{}) (res Result, err error) {
	j.job.Run(args...)
	return
}

type jobIndexed struct {
	job Job
}

func (j *jobIndexed) Run(workerIndex, subIndex int, args ...interface{}) (res Result, err error) {
	res, err = j.job.Run(args...)
	return
}
