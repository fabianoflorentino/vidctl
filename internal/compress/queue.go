package compress

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fabianoflorentino/vidctl/internal/dlog"
	"github.com/fabianoflorentino/vidctl/internal/events"
)

// Task states of the conversion queue.
const (
	StateQueued   = "queued"
	StateRunning  = "running"
	StateDone     = "done"
	StateError    = "error"
	StateCanceled = "canceled"
)

// Task is one unit of work waiting in the queue. Kind identifies the kind of
// work so later phases (audio extraction, transcription) share the same queue.
type Task struct {
	Kind  string
	JobID string
	Label string
	Job   Job
}

// TaskStatus is the snapshot of a task sent to the frontend.
type TaskStatus struct {
	JobID      string  `json:"jobId"`
	Kind       string  `json:"kind"`
	Label      string  `json:"label"`
	InputPath  string  `json:"inputPath"`
	PresetID   string  `json:"presetId"`
	State      string  `json:"state"`
	Position   int     `json:"position"`
	Percent    float64 `json:"percent"`
	Stage      string  `json:"stage"`
	OutputPath string  `json:"outputPath"`
	SizeBytes  int64   `json:"sizeBytes"`
	PartsTotal int     `json:"partsTotal"`
	PartsDone  int     `json:"partsDone"`
	Error      string  `json:"error"`
}

type taskEntry struct {
	task   Task
	state  string
	cancel context.CancelFunc

	percent    float64
	stage      string
	outputPath string
	sizeBytes  int64
	errMsg     string
	partsTotal int
	partsDone  int
}

// Manager runs queued compression jobs, up to maxParallel at a time, and
// tracks each task's status for the frontend.
type Manager struct {
	mu          sync.Mutex
	tasks       []*taskEntry
	running     int
	maxParallel int
	seq         int64
	run         func(ctx context.Context, jobID string, job Job) error
}

// NewManager creates a queue manager that runs one job at a time.
func NewManager() *Manager {
	return &Manager{
		maxParallel: 1,
		run:         Run,
	}
}

// Enqueue adds a task to the queue and returns its ID and position in the
// waiting line. It emits compress:queued before the worker may start, so the
// frontend registers the item first.
func (m *Manager) Enqueue(task Task) (string, int) {
	m.mu.Lock()
	if task.JobID == "" {
		m.seq++
		task.JobID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), m.seq)
	}
	if task.Kind == "" {
		task.Kind = events.KindCompress
	}
	e := &taskEntry{task: task, state: StateQueued}
	m.tasks = append(m.tasks, e)
	position := 1
	for _, prev := range m.tasks[:len(m.tasks)-1] {
		if prev.state == StateQueued || prev.state == StateRunning {
			position++
		}
	}
	m.mu.Unlock()

	dlog.Printf("[fila] enfileirado job=%s pos=%d input=%q", e.task.JobID, position, task.Job.InputPath)
	events.EmitQueued(e.task.JobID, position)
	m.pump()
	return e.task.JobID, position
}

// Cancel aborts a task: queued tasks leave the execution line, running tasks
// have their context canceled. Finished tasks are left untouched.
func (m *Manager) Cancel(id string) {
	dlog.Printf("[fila] cancelar pedido job=%s", id)
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.find(id)
	if e == nil {
		return
	}
	switch e.state {
	case StateQueued:
		e.state = StateCanceled
	case StateRunning:
		e.state = StateCanceled
		if e.cancel != nil {
			e.cancel()
		}
	}
}

// Remove forgets a task, whatever its state.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.tasks {
		if e.task.JobID == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			return
		}
	}
}

// ClearFinished drops every task that is no longer queued or running.
func (m *Manager) ClearFinished() {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.tasks[:0]
	for _, e := range m.tasks {
		if e.state == StateQueued || e.state == StateRunning {
			kept = append(kept, e)
		}
	}
	m.tasks = kept
}

// List returns a status snapshot of every task. Position is the number of
// unfinished tasks before it plus one, and zero for finished ones.
func (m *Manager) List() []TaskStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]TaskStatus, 0, len(m.tasks))
	pending := 0
	for _, e := range m.tasks {
		status := TaskStatus{
			JobID:      e.task.JobID,
			Kind:       e.task.Kind,
			Label:      e.task.Label,
			InputPath:  e.task.Job.InputPath,
			PresetID:   e.task.Job.PresetID,
			State:      e.state,
			Percent:    e.percent,
			Stage:      e.stage,
			OutputPath: e.outputPath,
			SizeBytes:  e.sizeBytes,
			PartsTotal: e.partsTotal,
			PartsDone:  e.partsDone,
			Error:      e.errMsg,
		}
		if e.state == StateQueued || e.state == StateRunning {
			pending++
			status.Position = pending
		}
		out = append(out, status)
	}
	return out
}

// SetMaxParallel changes how many jobs may run at once (values below one are
// clamped) and starts queued jobs when slots become available.
func (m *Manager) SetMaxParallel(n int) {
	if n < 1 {
		n = 1
	}
	dlog.Printf("[fila] maxParallel=%d", n)
	m.mu.Lock()
	m.maxParallel = n
	m.mu.Unlock()
	m.pump()
}

// Observe enriches the entry of a job with the payload of one of its
// compression events. Terminal states are never rewritten here.
func (m *Manager) Observe(name string, data any) {
	var jobID, kind string
	var progress *events.ProgressEvent
	var done *events.DoneEvent
	var failed *events.ErrorEvent
	switch ev := data.(type) {
	case events.ProgressEvent:
		jobID, kind, progress = ev.JobID, ev.Kind, &ev
	case events.DoneEvent:
		jobID, kind, done = ev.JobID, ev.Kind, &ev
	case events.ErrorEvent:
		jobID, kind, failed = ev.JobID, ev.Kind, &ev
	default:
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.find(jobID)
	if e == nil || e.state != StateRunning || e.task.Kind != kind {
		return
	}
	switch {
	case progress != nil:
		e.percent = progress.Percent
		e.stage = progress.Stage
		if _, total, ok := parseParts(progress.Stage); ok {
			e.partsTotal = total
		}
	case done != nil:
		e.outputPath = done.OutputPath
		e.sizeBytes += done.SizeBytes
		e.partsDone++
		e.percent = 100
		e.stage = "done"
	case failed != nil:
		e.errMsg = failed.Error
	}
}

// pump reserves free slots for queued tasks and starts them. Events are
// emitted only after the lock is released: the app emitter calls back into
// Observe, which takes the same mutex.
func (m *Manager) pump() {
	m.mu.Lock()
	type start struct {
		entry *taskEntry
		ctx   context.Context
	}
	var starts []start
	for _, e := range m.tasks {
		if m.running >= m.maxParallel {
			break
		}
		if e.state != StateQueued {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		e.cancel = cancel
		e.state = StateRunning
		m.running++
		starts = append(starts, start{entry: e, ctx: ctx})
	}
	m.mu.Unlock()

	for _, s := range starts {
		dlog.Printf("[fila] iniciando job=%s", s.entry.task.JobID)
		events.EmitStart(s.entry.task.JobID)
		go m.worker(s.ctx, s.entry)
	}
}

func (m *Manager) worker(ctx context.Context, e *taskEntry) {
	var err error
	if ctx.Err() == nil {
		err = m.run(ctx, e.task.JobID, e.task.Job)
	} else {
		err = ctx.Err()
	}

	m.mu.Lock()
	e.cancel = nil
	switch {
	case err == nil:
		e.state = StateDone
		e.percent = 100
		e.stage = "done"
	case e.state == StateCanceled:
	case errors.Is(err, context.Canceled) || ctx.Err() != nil:
		e.state = StateCanceled
	default:
		e.state = StateError
		e.errMsg = err.Error()
	}
	m.running--
	final := e.state
	m.mu.Unlock()

	dlog.Printf("[fila] worker fim job=%s estado=%s err=%v", e.task.JobID, final, err)
	m.pump()
}

// find returns the entry with the given ID. Callers must hold m.mu.
func (m *Manager) find(id string) *taskEntry {
	for _, e := range m.tasks {
		if e.task.JobID == id {
			return e
		}
	}
	return nil
}

// parseParts extracts the 1-based part index and total from a stage like
// "encoding · parte 2/3".
func parseParts(stage string) (index, total int, ok bool) {
	i := strings.Index(stage, "· parte ")
	if i < 0 {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(stage[i:], "· parte %d/%d", &index, &total); err != nil {
		return 0, 0, false
	}
	return index, total, true
}
