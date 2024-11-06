package taskmanager

import (
	"time"

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
	SavedCtx      []byte
}

// no RunTaskBefore()/RunTaskAfter(), please use TaskSteper
type Tasker interface {
	Id() string
	Name() string
	InitTaskStep(taskId, taskName string, input []byte) error
	RunTask() error
	GetRedoFlag() bool
	GetTask() *task
	ReloadCtx(oldCtx []byte) error // need InitTaskStep +  set redoSubStep
	Cancel()
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
	subStep       string
	redoSubStep   string // redo start point
	typ           string
	expiredAt     time.Time
}

func newTask(id, name string, canRedo bool, expiredAt time.Time) *task {
	t := &task{
		id:         id,
		name:       name,
		steps:      make([]TaskSteper, 0, 3),
		clearSteps: make([]TaskSteper, 0, 3),
		clearErrs:  make([]error, 0),
		canRedo:    canRedo,
		expiredAt:  expiredAt,
	}
	if expiredAt.IsZero() {
		t.expiredAt = time.Date(9999, 12, 31, 23, 59, 59, 0, time.Local)
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

func (t *task) Cancel() {
	log.Glog.Info("start to task", zap.String("id", t.id))

	if t.status == StatusDeleting {
		log.Glog.Info("task is deleting", zap.String("id", t.id))
		return
	}
	if t.status != StatusInProgress {
		log.Glog.Info("task isn't running", zap.String("id", t.id))
		return
	}

	t.exitFlag = true
	t.status = StatusAborted
	if t.err = t.UpdateTaskStatus(t.id, t.name, t.status); t.err != nil {
		return
	}
}

func (t *task) doClearStep(idx int) {
	if idx < 0 {
		log.Glog.Info("no task clear step when no start", zap.String("id", t.id), zap.String("name", t.name))
		return
	}
	if idx > len(t.steps)-1 {
		log.Glog.Warn("no task clear step when over steps", zap.String("id", t.id), zap.String("name", t.name))
		return
	}

	s := t.steps[idx].GetTaskStep()

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
}

func (t *task) doStep(idx int, st TaskSteper) {
	s := st.GetTaskStep()
	if t.redoSubStep != "" && s.name != t.redoSubStep {
		log.Glog.Info("skip task step for redo", zap.String("id", t.id), zap.String("name", t.name), zap.String("step", s.name), zap.String("redo_step", t.redoSubStep))

		return
	}

	log.Glog.Info("task step begin to run", zap.String("id", t.id), zap.String("name", t.name), zap.String("step", s.name))

	if t.err = t.UpdateSubStepTaskStatus(t.id, t.name, s.name, StatusInProgress); t.err != nil {
		return
	}

	t.err = st.Run()
	if t.err == nil {
		if t.err = t.UpdateSubStepTaskStatus(t.id, t.name, s.name, StatusCompleted); t.err != nil {
			return
		}
		return
	}

	t.status = StatusFailed
	if t.err = t.UpdateTaskStatus(t.id, t.name, t.status); t.err != nil {
		return
	}
}

func (t *task) RunTask() error {
	t.status = StatusInProgress
	if t.err = t.UpdateTaskStatus(t.id, t.name, t.status); t.err != nil {
		return t.err
	}

	var now time.Time
	for idx, sf := range t.steps {
		now = time.Now()
		if now.After(t.expiredAt) {
			log.Glog.Warn("expired task", zap.String("id", t.id), zap.Time("expiredAt", t.expiredAt))
			t.Cancel()
		}

		if t.exitFlag {
			log.Glog.Error("ExitFlag is configured in runtask", zap.String("id", t.id))
			t.doClearStep(idx - 1)
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
			log.Glog.Error("Excute task step failed", zap.String("id", t.id), zap.String("step", sf.GetTaskStep().name), zap.Error(t.err))
			t.doClearStep(idx)
			return t.err
		}

		t.progress += int(sf.GetTaskStep().ratio)
	}

	log.Glog.Info("run task finished", zap.String("id", t.id), zap.String("name", t.name))
	t.status = StatusCompleted
	t.progress = TaskComplete

	if t.err = t.UpdateTaskStatus(t.id, t.name, t.status); t.err != nil {
		return t.err
	}

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

func (t *task) UpdateSubStepTaskStatus(id, name, subStep string, subStepStatus int) error {
	if t.GetRedoFlag() {
		return manager.UpdateSubStepTaskStatus(id, name, subStep, subStepStatus)
	}

	return nil
}
