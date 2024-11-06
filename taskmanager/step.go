package taskmanager

import (
	"errors"
)

const (
	TaskStepSuffixClearRun = "ClearRun"
)

var (
	ErrNoStep = errors.New("no step")
)

type TaskSteper interface {
	Init()
	Run() error
	ClearRun() error
	GetTaskStep() *taskStep
}

type taskStep struct {
	tid string
	//sid        string
	name       string
	ratio      int
	order      int
	status     int
	progress   int
	expiration int
}

func newTaskStep(name string, ratio int) *taskStep {
	return &taskStep{
		//sid:    id.NewUUIDV4(true),
		name:   name,
		ratio:  ratio,
		status: StatusInitial,
	}
}

func (ts *taskStep) GetTaskStep() *taskStep {
	return ts
}
