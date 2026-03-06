package polling

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var ErrSkipTask = errors.New("skip task")

const SnapshotSettingKey = "observability.polling_snapshot"

type Task struct {
	Name       string
	Interval   time.Duration
	Timeout    time.Duration
	RunOnStart bool

	PreHandle  func(ctx context.Context) error
	Handle     func(ctx context.Context) error
	PostHandle func(ctx context.Context, runErr error) error
}

type TaskSnapshot struct {
	Name           string     `json:"name"`
	IntervalSecond int64      `json:"interval_second"`
	TimeoutSecond  int64      `json:"timeout_second"`
	RunOnStart     bool       `json:"run_on_start"`
	Running        bool       `json:"running"`
	LastStartedAt  *time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt *time.Time `json:"last_finished_at,omitempty"`
	LastDurationMS int64      `json:"last_duration_ms"`
	SuccessCount   int64      `json:"success_count"`
	FailureCount   int64      `json:"failure_count"`
	SkipCount      int64      `json:"skip_count"`
	LastError      string     `json:"last_error,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
}

type Snapshot struct {
	Started     bool           `json:"started"`
	TaskCount   int            `json:"task_count"`
	GeneratedAt time.Time      `json:"generated_at"`
	Tasks       []TaskSnapshot `json:"tasks"`
}

type taskStats struct {
	TaskSnapshot
}

type SnapshotPersister func(ctx context.Context, snapshot Snapshot) error

type Manager struct {
	mu      sync.RWMutex
	tasks   []Task
	started bool
	stats   map[string]*taskStats

	cancel context.CancelFunc
	wg     sync.WaitGroup

	logf            func(format string, args ...any)
	persistSnapshot SnapshotPersister
}

func NewManager() *Manager {
	return &Manager{
		logf:  log.Printf,
		stats: make(map[string]*taskStats),
	}
}

func (m *Manager) SetLogger(logf func(format string, args ...any)) {
	if logf == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logf = logf
}

func (m *Manager) SetSnapshotPersister(persist SnapshotPersister) {
	if persist == nil {
		return
	}

	m.mu.Lock()
	m.persistSnapshot = persist
	snapshot := m.snapshotLocked(time.Now())
	logf := m.logf
	m.mu.Unlock()

	m.publishSnapshot(snapshot, persist, logf)
}

func (m *Manager) Register(task Task) error {
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if task.Interval <= 0 {
		return fmt.Errorf("task interval must be greater than 0")
	}
	if task.Handle == nil {
		return fmt.Errorf("task handle is required")
	}

	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return fmt.Errorf("manager already started")
	}
	m.tasks = append(m.tasks, task)
	m.stats[task.Name] = &taskStats{TaskSnapshot: TaskSnapshot{
		Name:           task.Name,
		IntervalSecond: int64(task.Interval / time.Second),
		TimeoutSecond:  int64(task.Timeout / time.Second),
		RunOnStart:     task.RunOnStart,
	}}
	snapshot := m.snapshotLocked(time.Now())
	persist := m.persistSnapshot
	logf := m.logf
	m.mu.Unlock()

	m.publishSnapshot(snapshot, persist, logf)
	return nil
}

func (m *Manager) Start(parent context.Context) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return fmt.Errorf("manager already started")
	}
	ctx, cancel := context.WithCancel(parent)
	tasks := make([]Task, len(m.tasks))
	copy(tasks, m.tasks)
	m.cancel = cancel
	m.started = true
	logf := m.logf
	persist := m.persistSnapshot
	snapshot := m.snapshotLocked(time.Now())
	m.mu.Unlock()

	m.publishSnapshot(snapshot, persist, logf)

	for _, task := range tasks {
		t := task
		m.setNextRun(t.Name, time.Now())
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if t.RunOnStart {
				m.runTask(ctx, t)
			}

			ticker := time.NewTicker(t.Interval)
			defer ticker.Stop()
			m.setNextRun(t.Name, time.Now().Add(t.Interval))

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					m.setNextRun(t.Name, time.Now().Add(t.Interval))
					m.runTask(ctx, t)
				}
			}
		}()
	}

	if logf != nil {
		logf("polling manager started with %d tasks", len(tasks))
	}
	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return nil
	}
	cancel := m.cancel
	m.started = false
	m.cancel = nil
	logf := m.logf
	persist := m.persistSnapshot
	snapshot := m.snapshotLocked(time.Now())
	m.mu.Unlock()

	m.publishSnapshot(snapshot, persist, logf)

	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if logf != nil {
			logf("polling manager stopped")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) runTask(parent context.Context, task Task) {
	start := time.Now()
	m.markTaskStart(task.Name, start)
	runCtx := parent
	var cancel context.CancelFunc
	if task.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(parent, task.Timeout)
	} else {
		runCtx, cancel = context.WithCancel(parent)
	}
	defer cancel()

	runErr := error(nil)
	if task.PreHandle != nil {
		runErr = task.PreHandle(runCtx)
	}

	if runErr == nil {
		runErr = task.Handle(runCtx)
	}

	if task.PostHandle != nil {
		if postErr := task.PostHandle(runCtx, runErr); postErr != nil && runErr == nil {
			runErr = postErr
		}
	}

	m.mu.RLock()
	logf := m.logf
	m.mu.RUnlock()
	elapsed := time.Since(start)
	m.markTaskEnd(task.Name, runErr, elapsed)

	if logf == nil {
		return
	}

	switch {
	case runErr == nil:
		logf("polling task success: %s (%s)", task.Name, elapsed)
	case errors.Is(runErr, ErrSkipTask):
		logf("polling task skipped: %s (%s)", task.Name, elapsed)
	default:
		logf("polling task failed: %s (%s): %v", task.Name, elapsed, runErr)
	}
}

func (m *Manager) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshotLocked(time.Now())
}

func (m *Manager) snapshotLocked(generatedAt time.Time) Snapshot {
	out := Snapshot{
		Started:     m.started,
		TaskCount:   len(m.tasks),
		GeneratedAt: generatedAt,
		Tasks:       make([]TaskSnapshot, 0, len(m.tasks)),
	}
	for _, task := range m.tasks {
		stats, ok := m.stats[task.Name]
		if !ok || stats == nil {
			out.Tasks = append(out.Tasks, TaskSnapshot{
				Name:           task.Name,
				IntervalSecond: int64(task.Interval / time.Second),
				TimeoutSecond:  int64(task.Timeout / time.Second),
				RunOnStart:     task.RunOnStart,
			})
			continue
		}
		snapshot := stats.TaskSnapshot
		out.Tasks = append(out.Tasks, snapshot)
	}
	return out
}

func (m *Manager) setNextRun(name string, next time.Time) {
	m.mu.Lock()
	stats, ok := m.stats[name]
	if !ok || stats == nil {
		m.mu.Unlock()
		return
	}
	stats.NextRunAt = &next
	snapshot := m.snapshotLocked(time.Now())
	persist := m.persistSnapshot
	logf := m.logf
	m.mu.Unlock()

	m.publishSnapshot(snapshot, persist, logf)
}

func (m *Manager) markTaskStart(name string, startedAt time.Time) {
	m.mu.Lock()
	stats, ok := m.stats[name]
	if !ok || stats == nil {
		m.mu.Unlock()
		return
	}
	stats.Running = true
	stats.LastStartedAt = &startedAt
	stats.LastError = ""
	snapshot := m.snapshotLocked(time.Now())
	persist := m.persistSnapshot
	logf := m.logf
	m.mu.Unlock()

	m.publishSnapshot(snapshot, persist, logf)
}

func (m *Manager) markTaskEnd(name string, runErr error, elapsed time.Duration) {
	m.mu.Lock()
	stats, ok := m.stats[name]
	if !ok || stats == nil {
		m.mu.Unlock()
		return
	}
	finishedAt := time.Now()
	stats.Running = false
	stats.LastFinishedAt = &finishedAt
	stats.LastDurationMS = elapsed.Milliseconds()
	if runErr == nil {
		stats.SuccessCount++
		stats.LastError = ""
	} else if errors.Is(runErr, ErrSkipTask) {
		stats.SkipCount++
		stats.LastError = ""
	} else {
		stats.FailureCount++
		stats.LastError = runErr.Error()
	}
	snapshot := m.snapshotLocked(time.Now())
	persist := m.persistSnapshot
	logf := m.logf
	m.mu.Unlock()

	m.publishSnapshot(snapshot, persist, logf)
}

func (m *Manager) publishSnapshot(snapshot Snapshot, persist SnapshotPersister, logf func(format string, args ...any)) {
	if persist == nil {
		return
	}
	if err := persist(context.Background(), snapshot); err != nil && logf != nil {
		logf("polling snapshot persist failed: %v", err)
	}
}
