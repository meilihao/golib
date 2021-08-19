package job

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/meilihao/golib/v2/log"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

var (
	tracer trace.Tracer
	engine *xorm.Engine
)

func Init(e *xorm.Engine) {
	tracer = otel.Tracer("job")

	engine = e
}

type Job struct {
	Id int64
	// Uid         string
	ScheduleId     int64 // 非周期性调度id为0
	ScheduledCount int64 // 调度次数
	Name           string
	Remark         string
	NodeId         int64
	ScheduledAt    time.Time // 由Scheduler设置
	StartAt        time.Time
	EndAt          time.Time
	Req            jsoniter.RawMessage
	Result         string
	Status         string //  active、failed 和 succeed
	RetryMax       int32
	TryCount       int32
}

// 周期调度
// 一次性job直接下发, 不进入Scheduler
type Schedule struct {
	Id int64
	// Uid     string // uuid
	ResourceType int64 // = type
	ResourceId   string
	Name         string
	Remark       string
	OwnerId      int64
	Period       int64
	StartAt      time.Time
	EndAt        time.Time
	NextAt       int64 `xorm:"-"` //
	Timeout      int64 // 0不超时
	//CanConcurrent  bool
	LastTime  int64  // 避免mysql全0不让插入
	Count     int64  // 次数
	Status    string // active, stop
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time // deleted
}

func (s *Schedule) Stop(isDeleted bool) error {

	return nil
}

type JobList []*Schedule

func (s JobList) Len() int      { return len(s) }
func (s JobList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s JobList) Less(i, j int) bool {
	// Two zero times should return false.
	// Otherwise, zero is "greater" than any other time.
	// (To sort it at the end of the list.)
	if s[i].NextAt == 0 {
		return false
	}
	if s[j].NextAt == 0 {
		return true
	}
	return s[i].NextAt < s[j].NextAt
}

type Scheduler struct {
	cancelFn   func()
	List       JobList
	m          map[int64]int
	lock       *sync.RWMutex
	addChan    chan *Schedule
	removeChan chan int64
	fns        map[int64]JobDo
	running    bool
	jobWaiter  sync.WaitGroup // 避免因Stop()异步导致Schedule在应用退出时未即使处理
}

// 不使用指针: 避免上个任务还未结束, 新调度过来了
type JobDo func(s Schedule) error

// fns handler function
func NewScheduler(fns map[int64]JobDo) *Scheduler {
	js := &Scheduler{
		List:       make(JobList, 0, 10),
		m:          make(map[int64]int, 10),
		fns:        fns,
		lock:       new(sync.RWMutex),
		addChan:    make(chan *Schedule),
		removeChan: make(chan int64),
	}

	return js
}

func (js *Scheduler) Start() {
	js.lock.Lock()
	defer js.lock.Unlock()
	if js.running {
		return
	}

	var ctx context.Context
	ctx, js.cancelFn = context.WithCancel(context.Background())

	js.running = true
	go js.run(ctx)
}

func (js *Scheduler) run(ctx context.Context) {
	log.Glog.Info("job run")

	var now int64
	var tTime time.Time
	var timer *time.Timer = time.NewTimer(100000 * time.Hour)
	var fn JobDo

	for {
		now = time.Now().Unix()
		for _, v := range js.List {
			if v.LastTime == 0 || now-v.LastTime > v.Period {
				v.NextAt = now + v.Period
			} else {
				v.NextAt = v.LastTime + v.Period
			}
		}

		sort.Sort(js.List)

		// 设置下次唤醒时间
		if len(js.List) == 0 {
			timer.Reset(100000 * time.Hour)
		} else {
			timer.Reset(time.Duration(js.List[0].NextAt-now) * time.Second)
		}

		select {
		case tTime = <-timer.C:
			for _, v := range js.List {
				if v.EndAt.Unix() < now { // ended
					v.Stop(true)
				}
				if v.StartAt.Unix() > now { // not start
					continue
				}

				if fn = js.fns[v.ResourceType]; fn != nil {
					v.LastTime = tTime.Unix()
					v.Count++

					fn(*v)
				}
			}
		case <-ctx.Done():
			timer.Stop()

			for _, v := range js.List {
				v.Stop(false)

				js.jobWaiter.Done()
			}

			log.Glog.Info("job stop")

			return
		case entry := <-js.addChan:
			timer.Stop() // 新添加的可能是最近的

			if v, ok := js.m[entry.Id]; ok {
				entry.LastTime = js.List[v].LastTime
				entry.Count = js.List[v].Count

				js.List[v] = entry

				log.Glog.Info("job replace", zap.String("id", strconv.Itoa(int(entry.Id))))
			} else {
				js.List = append(js.List, entry)
				js.m[entry.Id] = len(js.List) - 1

				log.Glog.Info("job add", zap.String("id", strconv.Itoa(int(entry.Id))))
			}
		case id := <-js.removeChan:
			timer.Stop() // 删除的可能是当前timer依赖的那项
			js.removeSchedule(id)

			log.Glog.Info("job remove", zap.Int64("id", id))
		}
	}
}

// 重复添加即为替换
func (js *Scheduler) Add(entry *Schedule) int64 {
	js.lock.Lock()
	defer js.lock.Unlock()

	if !js.running {
		if v, ok := js.m[entry.Id]; ok {
			entry.LastTime = js.List[v].LastTime
			entry.Count = js.List[v].Count

			js.List[v] = entry
		} else {
			js.List = append(js.List, entry)
			js.m[entry.Id] = len(js.List) - 1

			js.jobWaiter.Add(1)
		}
	} else {
		js.addChan <- entry

		js.jobWaiter.Add(1)
	}
	return entry.Id
}

func (js *Scheduler) Stop() {
	js.lock.Lock()
	defer js.lock.Unlock()

	if js.running {
		js.cancelFn()
		js.jobWaiter.Wait()
		js.running = false
	}
}

func (js *Scheduler) Remove(id int64) {
	js.lock.Lock()
	defer js.lock.Unlock()

	if js.running {
		js.removeChan <- id
	} else {
		js.removeSchedule(id)
	}

	js.jobWaiter.Done()
}

func (js *Scheduler) removeSchedule(id int64) {
	var ls []*Schedule
	for _, v := range js.List {
		if v.Id != id {
			ls = append(ls, v)
		} else {
			v.Stop(v.Id < 0) // 负数是彻底删除

			if v.Id < 0 {
				delete(js.m, -id)
			} else {
				delete(js.m, id)
			}
		}
	}

	js.List = JobList(ls)
}
