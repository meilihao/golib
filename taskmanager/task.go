package taskmanager

import (
	"github.com/meilihao/golib/v2/log"
	"go.uber.org/zap"
)

const (
	StatusNoExists = iota
	StatusInitial
	StatusInProgress
	StatusInProgressMovingDBF
	StatusFailed
	StatusCompleted
	StatusDeleting
	StatusDeleted
	StatusUpdating
	StatusAborted
	StatusFailedClearing
)

const (
	TaskComplete = 100
)

type TaskInfo struct {
	Id            string
	Status        int
	Name          string
	SubStepStatus int
	SubStepName   string
	StartTime     int64
	Typ           string
	Input         []byte
}

// no RunTaskBefore()/RunTaskAfter(), please use TaskSteper
type Tasker interface {
	Id() string
	Name() string
	InitTaskStep(input []byte) error
	RunTask() error
	GetRedoFlag() bool
	GetTask() *task
}

type task struct {
	id            string
	name          string
	input         []byte
	status        int
	steps         []TaskSteper
	clearSteps    []TaskSteper
	clearErrs     []error
	progress      int
	exitFlag      bool // must set err too
	err           error
	canRedo       bool // task support to redo
	subStepStatus string
	subStepName   string
	typ           string
}

func newTask(id, name string, canRedo bool) *task {
	t := &task{
		id:         id,
		name:       name,
		steps:      make([]TaskSteper, 0, 3),
		clearSteps: make([]TaskSteper, 0, 3),
		clearErrs:  make([]error, 0),
		canRedo:    canRedo,
	}

	return t
}

func (t *task) Id() string {
	return t.id
}

func (t *task) Name() string {
	return t.name
}

func (t *task) GetRedoFlag() bool {
	return t.canRedo
}

func (t *task) GetTask() *task {
	return t
}

func (t *task) doStep(idx int, st TaskSteper) {
	s := st.GetTaskStep()
	log.Glog.Info("task step begin to run", zap.String("id", t.id), zap.String("name", t.name), zap.String("step", s.name))

	// todo: UpdateSubStepTask

	t.err = st.Run()
	if t.err == nil {
		// todo: UpdateSubStepTask
		return
	}
	log.Glog.Error("Execute task step fail", zap.String("id", t.id), zap.String("name", t.name), zap.String("step", s.name), zap.Error(t.err))

	clearSteps := t.clearSteps
	if len(clearSteps) > 0 {
		t.status = StatusFailedClearing

		log.Glog.Info("Execute task step clear start", zap.String("id", t.id), zap.String("name", t.name), zap.String("step", s.name))

		var cErr error
		var cSteper TaskSteper
		var cSetp *taskStep
		for i := idx; i >= 0; i-- {
			cSteper = t.clearSteps[i]
			cSetp = cSteper.GetTaskStep()

			log.Glog.Info("Execute task step clearing", zap.String("id", t.id), zap.String("name", t.name), zap.String("step", cSetp.name+"@"+TaskStepSuffixClearRun))

			if cErr = cSteper.ClearRun(); cErr != nil {
				log.Glog.Error("Execute task step clear failed", zap.String("id", t.id), zap.String("name", t.name), zap.String("step", cSetp.name+"@"+TaskStepSuffixClearRun), zap.Error(cErr))

				t.clearErrs = append(t.clearErrs, cErr)
			}
		}

		log.Glog.Info("Execute task step clear end", zap.String("id", t.id), zap.String("name", t.name), zap.String("step", s.name))
	}

	t.status = StatusFailed
	// todo: UpdateTaskStatus
}

func (t *task) RunTask() error {
	t.status = StatusInProgress
	if t.err = t.UpdateTaskStatus(t.id, t.name, t.status); t.err != nil {
		return t.err
	}

	for idx, sf := range t.steps {
		if t.exitFlag {
			log.Glog.Error("ExitFlag is configured in runtask", zap.String("id", t.id))
			t.status = StatusAborted
			return t.err
		}

		if sf == nil {
			log.Glog.Error("Unexpected exception, task step is NULL", zap.String("id", t.id))
			t.err = ErrNoStep
			t.status = StatusFailed
			return t.err
		}

		t.doStep(idx, sf)
		if t.err != nil {
			//log.Glog.Error("Excute task step failed", zap.String("id", t.id), zap.String("step", sf.GetTaskStep().name), zap.Error(t.err))
			return t.err
		}

		t.progress += int(sf.GetTaskStep().ratio)
	}

	log.Glog.Info("run task finished", zap.String("id", t.id), zap.String("name", t.name))
	t.status = StatusCompleted
	t.progress = TaskComplete

	//  todo: UpdateTaskStatus

	return t.err
}

func (t *task) addStep(steper TaskSteper) {
	ts := steper.GetTaskStep()
	if ts.tid == "" {
		ts.tid = t.id
	}
	ts.order = len(t.steps) + 1

	t.steps = append(t.steps, steper)
	t.clearSteps = append(t.clearSteps, steper)
}

func (t *task) UpdateTaskStatus(id, name string, status int) error {
	if t.GetRedoFlag() {
		return manager.UpdateTaskStatus(id, name, status)
	}

	return nil
}
