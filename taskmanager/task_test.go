package taskmanager

import (
	"fmt"
	"testing"
	"time"
)

var (
	_ Tasker = new(DemoTask)
)

type DemoTask struct {
	*task
}

func (dt *DemoTask) InitTaskStep(taskId, taskName string, input []byte) error {
	dt.task = newTask(taskId, taskName, false, time.Time{})
	dt.CreateTaskStep()

	return nil
}

func (dt *DemoTask) ReloadCtx(oldCtx []byte) error {
	return nil
}

func (dt *DemoTask) CreateTaskStep() {
	dt.addStep(NewTaskStepDemoInit(50))
	dt.addStep(NewTaskStepDemoDone(100))
}

type TaskStepDemoInit struct {
	*taskStep
}

func NewTaskStepDemoInit(ratio int) *TaskStepDemoInit {
	s := &TaskStepDemoInit{
		taskStep: newTaskStep("TaskStepDemoInit", ratio),
	}

	return s
}

func (tsDI *TaskStepDemoInit) Init() {

}

func (tsDI *TaskStepDemoInit) Run() error {
	return nil
}

func (tsDI *TaskStepDemoInit) ClearRun() error {
	return nil
}

type TaskStepDemoDone struct {
	*taskStep
}

func NewTaskStepDemoDone(ratio int) *TaskStepDemoDone {
	s := &TaskStepDemoDone{
		taskStep: newTaskStep("TaskStepDemoDone", ratio),
	}

	return s
}

func (tsDd *TaskStepDemoDone) Init() {

}

func (tsDd *TaskStepDemoDone) Run() error {
	return fmt.Errorf("err")
}

func (tsDd *TaskStepDemoDone) ClearRun() error {
	return nil
}

func TestTask(t *testing.T) {
	task := new(DemoTask)
	RunSyncTask(task, "1", "demo", nil)
}
