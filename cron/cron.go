package cron

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/meilihao/golib/v2/log"
	"go.uber.org/zap"
)

// Cron keeps track of any number of entries, invoking the associated func as
// specified by the schedule. It may be started, stopped, and the entries may
// be inspected while running.
type Cron struct {
	entries   []*Entry
	chain     Chain
	stop      chan struct{}
	add       chan *Entry
	remove    chan EntryID
	snapshot  chan chan []Entry
	running   bool
	logger    Logger
	runningMu sync.Mutex
	location  *time.Location
	parser    ScheduleParser
	nextID    EntryID
	jobWaiter sync.WaitGroup
}

// ScheduleParser is an interface for schedule spec parsers that return a Schedule
type ScheduleParser interface {
	Parse(spec string) (Schedule, error)
}

// Job is an interface for submitted cron jobs.
type Job interface {
	Run()
	ID() string
	Next(time.Time) time.Time
	EndTime() time.Time
}

// Schedule describes a job's duty cycle.
type Schedule interface {
	// Next returns the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(time.Time) time.Time
}

// EntryID identifies an entry within a Cron instance
type EntryID int

// Entry consists of a schedule and the func to execute on that schedule.
type Entry struct {
	// ID is the cron-assigned ID of this entry, which may be used to look up a
	// snapshot or remove it.
	ID       EntryID
	UniqueID string

	// Schedule on which this job should be run.
	Schedule Schedule

	// Next time the job will run, or the zero time if Cron has not been
	// started or this entry's schedule is unsatisfiable
	Next time.Time

	// Prev is the last time this job was run, or the zero time if never.
	Prev time.Time

	// WrappedJob is the thing to run when the Schedule is activated.
	WrappedJob Job

	// Job is the thing that was submitted to cron.
	// It is kept around so that user code that needs to get at the job later,
	// e.g. via Entries() can do so.
	Job    Job
	Status int16 // -1: over end time
}

func (e *Entry) NextSchedule(now time.Time) {
	e.Next = e.Job.Next(now)
	if e.Next.IsZero() {
		e.Next = e.Schedule.Next(now)
	}
}

// Valid returns true if this is not the zero entry.
func (e Entry) Valid() bool { return e.ID != 0 }

// byTime is a wrapper for sorting the entry array by time
// (with zero time at the end).
type byTime []*Entry

func (s byTime) Len() int      { return len(s) }
func (s byTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byTime) Less(i, j int) bool {
	// Two zero times should return false.
	// Otherwise, zero is "greater" than any other time.
	// (To sort it at the end of the list.)
	if s[i].Next.IsZero() {
		return false
	}
	if s[j].Next.IsZero() {
		return true
	}
	return s[i].Next.Before(s[j].Next)
}

// New returns a new Cron job runner, modified by the given options.
//
// Available Settings
//
//   Time Zone
//     Description: The time zone in which schedules are interpreted
//     Default:     time.Local
//
//   Parser
//     Description: Parser converts cron spec strings into cron.Schedules.
//     Default:     Accepts this spec: https://en.wikipedia.org/wiki/Cron
//
//   Chain
//     Description: Wrap submitted jobs to customize behavior.
//     Default:     A chain that recovers panics and logs them to stderr.
//
// See "cron.With*" to modify the default behavior.
func New(opts ...Option) *Cron {
	c := &Cron{
		entries:   nil,
		chain:     NewChain(),
		add:       make(chan *Entry),
		stop:      make(chan struct{}),
		snapshot:  make(chan chan []Entry),
		remove:    make(chan EntryID),
		running:   false,
		runningMu: sync.Mutex{},
		logger:    DefaultLogger,
		location:  time.Local,
		parser:    standardParser,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// FuncJob is a wrapper that turns a func() into a cron.Job
type FuncJob func()

func (f FuncJob) Run() { f() }

func (f FuncJob) ID() string                   { return "" }
func (f FuncJob) Next(now time.Time) time.Time { return time.Time{} }

func (f FuncJob) EndTime() time.Time { return time.Date(9999, 1, 1, 0, 0, 0, 0, time.Local) }

// AddFunc adds a func to the Cron to be run on the given schedule.
// The spec is parsed using the time zone of this Cron instance as the default.
// An opaque ID is returned that can be used to later remove it.
func (c *Cron) AddFunc(spec string, cmd func()) (EntryID, error) {
	return c.AddJob(spec, FuncJob(cmd))
}

// AddJob adds a Job to the Cron to be run on the given schedule.
// The spec is parsed using the time zone of this Cron instance as the default.
// An opaque ID is returned that can be used to later remove it.
func (c *Cron) AddJob(spec string, cmd Job) (EntryID, error) {
	schedule, err := c.parser.Parse(spec)
	if err != nil {
		return 0, err
	}
	return c.Schedule(schedule, cmd), nil
}

// Schedule adds a Job to the Cron to be run on the given schedule.
// The job is wrapped with the configured Chain.
func (c *Cron) Schedule(schedule Schedule, cmd Job) EntryID {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	c.nextID++
	entry := &Entry{
		ID:         c.nextID,
		UniqueID:   cmd.ID(),
		Schedule:   schedule,
		WrappedJob: c.chain.Then(cmd),
		Job:        cmd,
	}
	if !c.running {
		c.entries = append(c.entries, entry)
	} else {
		c.add <- entry
	}
	return entry.ID
}

// Entries returns a snapshot of the cron entries.
func (c *Cron) Entries() []Entry {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	if c.running {
		replyChan := make(chan []Entry, 1)
		c.snapshot <- replyChan
		return <-replyChan
	}
	return c.entrySnapshot()
}

// Location gets the time zone location
func (c *Cron) Location() *time.Location {
	return c.location
}

// Entry returns a snapshot of the given entry, or nil if it couldn't be found.
func (c *Cron) Entry(id EntryID) Entry {
	for _, entry := range c.Entries() {
		if id == entry.ID {
			return entry
		}
	}
	return Entry{}
}

// Remove an entry from being run in the future.
func (c *Cron) Remove(id EntryID) {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	if c.running {
		c.remove <- id
	} else {
		c.removeEntry(id)
	}
}

// GetEntryBySID get an entry by sid.
func (c *Cron) GetEntryBySID(sid string) Entry {
	for _, v := range c.entries {
		if v.Job.ID() == sid {
			return *v
		}
	}

	return Entry{}
}

// GetEntry get an entry by id.
func (c *Cron) GetEntry(id EntryID) Entry {
	for _, v := range c.entries {
		if v.ID == id {
			return *v
		}
	}

	return Entry{}
}

// Remove an entry from being run in the future by sid.
func (c *Cron) RemoveBySID(sid string) {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()

	e := c.GetEntryBySID(sid)
	if e.ID == 0 {
		return
	}

	if c.running {
		c.remove <- e.ID
	} else {
		c.removeEntry(e.ID)
	}
}

// Start the cron scheduler in its own goroutine, or no-op if already started.
func (c *Cron) Start() {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	if c.running {
		return
	}
	c.running = true
	go c.run()
}

// Run the cron scheduler, or no-op if already running.
func (c *Cron) Run() {
	c.runningMu.Lock()
	if c.running {
		c.runningMu.Unlock()
		return
	}
	c.running = true
	c.runningMu.Unlock()
	c.run()
}

// run the scheduler.. this is private just due to the need to synchronize
// access to the 'running' state variable.
func (c *Cron) run() {
	log.Glog.Info("cron start")

	// Figure out the next activation times for each entry.
	now := c.now()
	for _, entry := range c.entries {
		entry.NextSchedule(now)
		log.Glog.Debug("cron schedule first", zap.Time("now", now), zap.Time("next", entry.Next), zap.String("unique_id", entry.UniqueID), zap.Int("len", len(c.entries)))
	}

	for {
		// Determine the next entry to run.
		sort.Sort(byTime(c.entries))

		var timer *time.Timer
		for _, entry := range c.entries {
			// log.Glog.Debug("cron schedule range", zap.Time("now", now),
			// 	zap.Time("next", entry.Next), zap.String("end_time", entry.Job.EndTime().String()),
			// 	zap.Bool("c1", !entry.Job.EndTime().IsZero() && entry.Next.After(entry.Job.EndTime())),
			// 	zap.Bool("c2", entry.Next.Before(now)),
			// 	zap.String("unique_id", entry.UniqueID))
			if (!entry.Job.EndTime().IsZero() && entry.Next.After(entry.Job.EndTime())) || entry.Next.Before(now) { //  entry.Next > entry.Job.EndTime() || entry.Next > now
				if entry.Status < 0 {
					continue
				}

				entry.Status = -1

				go func(id EntryID) {
					c.remove <- id
				}(entry.ID)

				log.Glog.Debug("cron schedule end", zap.Int("id", int(entry.ID)), zap.Time("now", now), zap.Time("next", entry.Next), zap.String("end_time", entry.Job.EndTime().String()), zap.String("unique_id", entry.UniqueID))
			} else {
				timer = time.NewTimer(entry.Next.Sub(now))
				log.Glog.Debug("cron schedule next", zap.Int("id", int(entry.ID)), zap.Time("now", now), zap.Time("next", entry.Next), zap.Duration("duration", entry.Next.Sub(now)), zap.String("unique_id", entry.UniqueID))
				break
			}
		}
		if timer == nil {
			// If there are no entries yet, just sleep - it still handles new entries
			// and stop requests.
			timer = time.NewTimer(100000 * time.Hour)
		}

		for {
			select {
			case now = <-timer.C:
				now = now.In(c.location)
				log.Glog.Debug("cron wake", zap.Time("now", now))

				// Run every entry whose next time was less than now
				for _, e := range c.entries {
					if e.Next.After(now) || e.Next.IsZero() {
						break
					}
					if e.Status < 0 {
						continue
					}

					c.startJob(e.WrappedJob)
					e.Prev = e.Next
					e.NextSchedule(now)
					log.Glog.Info("cron do", zap.Int("id", int(e.ID)), zap.Time("now", now), zap.Time("next", e.Next), zap.String("unique_id", e.UniqueID))
				}

			case newEntry := <-c.add:
				timer.Stop() // new job may be schedule first
				now = c.now()
				newEntry.NextSchedule(now)
				c.entries = append(c.entries, newEntry)
				log.Glog.Info("cron added", zap.Int("id", int(newEntry.ID)), zap.Time("now", now), zap.Time("next", newEntry.Next), zap.String("unique_id", newEntry.UniqueID))

			case replyChan := <-c.snapshot:
				replyChan <- c.entrySnapshot()
				continue

			case <-c.stop:
				timer.Stop()
				log.Glog.Info("cron stop")
				return

			case id := <-c.remove:
				if entry := c.GetEntry(id); entry.Valid() {
					timer.Stop()  // timer may be set by this entry
					now = c.now() // for test TestScheduleAfterRemoval
					c.removeEntry(id)
					log.Glog.Info("cron removed", zap.Int("id", int(entry.ID)), zap.String("unique_id", entry.UniqueID))
				}
			}

			break
		}
	}
}

// startJob runs the given job in a new goroutine.
func (c *Cron) startJob(j Job) {
	c.jobWaiter.Add(1)
	go func() {
		defer c.jobWaiter.Done()
		j.Run()
	}()
}

// now returns current time in c location
func (c *Cron) now() time.Time {
	return time.Now().In(c.location)
}

// Stop stops the cron scheduler if it is running; otherwise it does nothing.
// A context is returned so the caller can wait for running jobs to complete.
func (c *Cron) Stop() context.Context {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	if c.running {
		c.stop <- struct{}{}
		c.running = false
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c.jobWaiter.Wait()
		cancel()
	}()
	return ctx
}

// entrySnapshot returns a copy of the current cron entry list.
func (c *Cron) entrySnapshot() []Entry {
	var entries = make([]Entry, len(c.entries))
	for i, e := range c.entries {
		entries[i] = *e
	}
	return entries
}

func (c *Cron) removeEntry(id EntryID) {
	var entries []*Entry
	for _, e := range c.entries {
		if e.ID != id {
			entries = append(entries, e)
		}
	}
	c.entries = entries
}
