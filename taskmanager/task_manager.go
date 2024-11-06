package taskmanager

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/meilihao/golib/v2/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	DBJobs = "Jobs"
)

var (
	manager *TaskManager
)

type TaskManager struct {
	db                *gorm.DB
	lock              sync.RWMutex
	pool              map[string]Tasker
	redoFuncContainer map[string]reflect.Type
}

func NewTaskManager(db *gorm.DB) *TaskManager {
	m := &TaskManager{
		db:                db,
		pool:              make(map[string]Tasker, 64),
		redoFuncContainer: make(map[string]reflect.Type),
	}

	return m
}

func (m *TaskManager) RegiesterRedoTasker(n string, i any) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if other := m.redoFuncContainer[n]; other != nil {
		panic("register double redo task: " + n)
	}
	t := reflect.TypeOf(i)

	m.redoFuncContainer[n] = t
}

func (m *TaskManager) AddTask(er Tasker) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	tid := er.GetTask().Id()
	if other := m.pool[tid]; other != nil {
		return fmt.Errorf("double task(%s)", tid)
	}

	m.pool[tid] = er

	log.Glog.Info("added task", zap.String("id", tid))

	return nil
}

func (m *TaskManager) Cancel(id string) error {
	m.lock.Lock()

	er := m.pool[id]
	if er == nil {
		m.lock.Unlock()
		return fmt.Errorf("no task(%s) to cancel", id)
	}
	m.lock.Unlock()

	er.Cancel()

	return nil
}

func RunSyncSubTask(t Tasker, input []byte) error {
	info, err := manager.QueryTaskInfo(t.Id(), t.Name())
	if err != nil {
		log.Glog.Error("get task info failed", zap.String("id", t.Id()), zap.Error(err))

		return err
	}

	if info.Id == "" {
		info.Id = t.Id()
		info.Status = StatusInitial
		info.Name = t.Name()
		info.Input = input

		if err = manager.SaveTask(t, info); err != nil {
			log.Glog.Error("save task info failed", zap.String("id", t.Id()), zap.Error(err))

			return err
		}
	}

	return nil
}

func RunSyncTask(t Tasker, taskId, taskName string, input []byte) error {
	log.Glog.Info("task run sync task", zap.String("id", taskId), zap.String("name", taskName))

	err := RunSyncSubTask(t, input)
	if err != nil {
		log.Glog.Error("task save task info failed", zap.String("id", taskId), zap.Error(err))

		return err
	}

	err = t.InitTaskStep(taskId, taskName, input)
	if err != nil {
		log.Glog.Error("Init sync task step failed", zap.String("id", taskId), zap.Error(err))
	} else {
		err = t.RunTask()
	}

	return err
}

func (m *TaskManager) DeleteExpireTasks() error {
	var expiredTimeout int64 = 3600 * 24
	now := time.Now().Unix() - expiredTimeout

	return m.db.Exec(fmt.Sprintf("delete from %s where startTime < ?", DBJobs), now).Error
}

func (m *TaskManager) SaveTask(t Tasker, info *TaskInfo) error {
	if !t.GetRedoFlag() {
		log.Glog.Warn("It's not redo task, don't insert into database", zap.String("id", t.Id()), zap.String("name", t.Name()))
		return nil
	}

	err := m.DeleteExpireTasks()
	if err != nil {
		log.Glog.Error("Delete expire tasks failed", zap.String("id", t.Id()), zap.Error(err))
	}

	log.Glog.Info("Begin save task info to db")

	sqlStr := fmt.Sprintf("insert into %s (Id,Status,Name,SubStepStatus, SubStepName, Input, Typ, StartTime) values(?,?,?,?,?,?,?,?)", DBJobs)
	// todo: EncryptInput

	it := t.GetTask()
	err = m.db.Exec(sqlStr, it.id, it.status, it.name, it.subStepStatus, it.subStep, it.input, it.typ, time.Now().Unix()).Error
	if err != nil {
		log.Glog.Info("save task info failed", zap.String("id", t.Id()), zap.Error(err))
		return err
	}

	log.Glog.Info("save task info success", zap.String("id", t.Id()))
	return nil
}

func (m *TaskManager) QueryTaskInfo(id, name string) (*TaskInfo, error) {
	sqlStr := fmt.Sprintf("select Id, Status, Name, SubStepStatus, SubStepName, Input, Typ from %s where Id = ? and Name = ?", DBJobs)

	info := new(TaskInfo)
	err := m.db.Raw(sqlStr, id, name).Scan(info).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Glog.Error("sqlDB.QueryTable failed", zap.Error(err))
		return nil, err
	}

	return info, nil
}

func (m *TaskManager) UpdateSubStepTaskStatus(id, name, subStep string, subStepStatus int) error {
	log.Glog.Info("Begin to update step task status", zap.String("id", id), zap.String("name", name), zap.String("step", subStep))

	sqlStr := fmt.Sprintf("update %s set SubStep = ?, SubStepStatus=? where Id = ? and Name = ?", DBJobs)

	err := m.db.Exec(sqlStr, subStep, subStepStatus, id, name).Error
	if err != nil {
		log.Glog.Error("Update step task status failed", zap.String("id", id), zap.String("name", name), zap.String("step", subStep), zap.Error(err))
		return err
	}
	log.Glog.Debug("Update step task status succ", zap.String("id", id), zap.String("name", name), zap.String("step", subStep))
	return nil
}

func (m *TaskManager) UpdateTaskStatus(id, name string, status int) error {
	log.Glog.Info("Begin to update task status", zap.String("id", id), zap.String("name", name), zap.Int("status", status))

	sqlStr := fmt.Sprintf("update %s set Status=? where Id = ? and Name = ?", DBJobs)

	err := m.db.Exec(sqlStr, status, id, name).Error
	if err != nil {
		log.Glog.Error("Update task status failed", zap.String("id", id), zap.String("name", name), zap.Int("status", status), zap.Error(err))
		return err
	}
	log.Glog.Debug("Update task status succ", zap.String("id", id), zap.String("name", name), zap.Int("status", status))
	return nil
}

func (m *TaskManager) GetAllRunningFromDB() ([]*TaskInfo, error) {
	sqlStr := fmt.Sprintf("select Id,Name,SavedCtx from %s where Status = ? and SubStepStatus = ?", DBJobs)

	ls := make([]*TaskInfo, 0)
	err := m.db.Raw(sqlStr, StatusInProgress, StatusInProgress).Scan(&ls).Error
	if err != nil {
		log.Glog.Error("sqlDB.QueryTable failed", zap.Error(err))
		return nil, err
	}

	log.Glog.Info("some task is still running, will do again", zap.Int("num", len(ls)))

	return ls, nil
}

func (m *TaskManager) CreateRedoTask() error {
	ls, err := m.GetAllRunningFromDB()
	if err != nil {
		return err
	}

	for _, ti := range ls {
		log.Glog.Debug("recreate task", zap.String("name", ti.Name), zap.String("step", ti.SubStepName))

		tpy := m.redoFuncContainer[ti.Name]
		if tpy == nil {
			log.Glog.Error("no redo task type", zap.String("name", ti.Name))
			continue
		}

		rter := reflect.New(tpy)
		rt := rter.Interface().(Tasker)
		if err = rt.ReloadCtx(ti.SavedCtx); err != nil {
			log.Glog.Error("redo task reload savedCtx", zap.String("name", ti.Name), zap.String("savedCtx", string(ti.SavedCtx)))
			continue
		}

		if err = m.AddTask(rt); err != nil {
			log.Glog.Error("add redo task", zap.String("name", ti.Name))
			continue
		}

		go func(name string) {
			log.Glog.Info("redo task start", zap.String("name", name))
			if err := rt.RunTask(); err != nil {
				log.Glog.Error("redo task", zap.String("name", name), zap.Error(err))
			}
			log.Glog.Info("redo task end", zap.String("name", name))
		}(ti.Name)
	}

	return nil
}
