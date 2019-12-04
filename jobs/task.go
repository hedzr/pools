// Copyright © 2019 Hedzr Yeh.

package jobs

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
