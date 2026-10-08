package compress

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fabianoflorentino/vidctl/internal/events"
)

func statusByID(m *Manager, id string) (TaskStatus, bool) {
	for _, s := range m.List() {
		if s.JobID == id {
			return s, true
		}
	}
	return TaskStatus{}, false
}

func waitState(t *testing.T, m *Manager, id, want string) TaskStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last TaskStatus
	for time.Now().Before(deadline) {
		s, ok := statusByID(m, id)
		if ok {
			last = s
			if s.State == want {
				return s
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("task %s ficou em %q (%+v), esperava %q", id, last.State, last, want)
	return last
}

func TestEnqueueRunsInFIFOOrder(t *testing.T) {
	release := make(chan struct{})
	started := make(chan string, 3)
	var mu sync.Mutex
	var order []string

	m := NewManager()
	m.run = func(_ context.Context, jobID string, _ Job) error {
		started <- jobID
		<-release
		mu.Lock()
		order = append(order, jobID)
		mu.Unlock()
		return nil
	}

	id1, pos1 := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	id2, pos2 := m.Enqueue(Task{Job: Job{InputPath: "b.mp4"}})
	id3, pos3 := m.Enqueue(Task{Job: Job{InputPath: "c.mp4"}})

	if pos1 != 1 || pos2 != 2 || pos3 != 3 {
		t.Errorf("posições do enqueue = %d,%d,%d; queria 1,2,3", pos1, pos2, pos3)
	}
	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Errorf("IDs deveriam ser únicos: %q %q %q", id1, id2, id3)
	}

	select {
	case got := <-started:
		if got != id1 {
			t.Errorf("primeiro a rodar = %q; queria %q", got, id1)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("primeiro job não iniciou")
	}

	s2, _ := statusByID(m, id2)
	s3, _ := statusByID(m, id3)
	if s2.State != StateQueued || s3.State != StateQueued {
		t.Errorf("estados = %q/%q; queria queued/queued", s2.State, s3.State)
	}
	if s2.Position != 2 || s3.Position != 3 {
		t.Errorf("posições na lista = %d,%d; queria 2,3 (id1 conta como não finalizada)", s2.Position, s3.Position)
	}

	close(release)
	waitState(t, m, id1, StateDone)
	waitState(t, m, id2, StateDone)
	waitState(t, m, id3, StateDone)

	mu.Lock()
	got := append([]string(nil), order...)
	mu.Unlock()
	if len(got) != 3 || got[0] != id1 || got[1] != id2 || got[2] != id3 {
		t.Errorf("ordem de execução = %v; queria [%s %s %s]", got, id1, id2, id3)
	}

	s1, _ := statusByID(m, id1)
	if s1.Position != 0 {
		t.Errorf("tarefa concluída com position %d; queria 0", s1.Position)
	}
	if s1.Percent != 100 || s1.Stage != "done" {
		t.Errorf("tarefa concluída = {%%:%v stage:%q}; queria 100/done", s1.Percent, s1.Stage)
	}
}

func TestCancelQueuedTaskNeverRuns(t *testing.T) {
	release := make(chan struct{})
	ran := make(chan string, 2)
	m := NewManager()
	m.run = func(_ context.Context, jobID string, _ Job) error {
		ran <- jobID
		<-release
		return nil
	}

	id1, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	id2, _ := m.Enqueue(Task{Job: Job{InputPath: "b.mp4"}})
	<-ran

	m.Cancel(id2)
	if s := waitState(t, m, id2, StateCanceled); s.Position != 0 {
		t.Errorf("cancelada com position %d; queria 0", s.Position)
	}

	close(release)
	waitState(t, m, id1, StateDone)
	waitState(t, m, id2, StateCanceled)

	select {
	case got := <-ran:
		t.Errorf("tarefa cancelada chegou a rodar: %q", got)
	default:
	}
}

func TestCancelRunningTaskStopsRun(t *testing.T) {
	started := make(chan struct{})
	m := NewManager()
	m.run = func(ctx context.Context, _ string, _ Job) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}

	id, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("job não iniciou")
	}

	m.Cancel(id)
	waitState(t, m, id, StateCanceled)
}

func TestSetMaxParallel(t *testing.T) {
	release := make(chan struct{})
	started := make(chan string, 4)
	m := NewManager()
	m.run = func(_ context.Context, jobID string, _ Job) error {
		started <- jobID
		<-release
		return nil
	}

	id1, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	id2, _ := m.Enqueue(Task{Job: Job{InputPath: "b.mp4"}})

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("primeiro job não iniciou")
	}
	if s, _ := statusByID(m, id2); s.State != StateQueued {
		t.Errorf("segundo job em %q; com maxParallel=1 deveria estar queued", s.State)
	}

	m.SetMaxParallel(2)
	select {
	case got := <-started:
		if got != id2 {
			t.Errorf("segundo a rodar = %q; queria %q", got, id2)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("segundo job não iniciou após SetMaxParallel(2)")
	}

	close(release)
	waitState(t, m, id1, StateDone)
	waitState(t, m, id2, StateDone)
}

func TestSetMaxParallelClampsToOne(t *testing.T) {
	release := make(chan struct{})
	started := make(chan string, 2)
	m := NewManager()
	m.run = func(_ context.Context, jobID string, _ Job) error {
		started <- jobID
		<-release
		return nil
	}

	m.SetMaxParallel(0)
	id1, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	id2, _ := m.Enqueue(Task{Job: Job{InputPath: "b.mp4"}})

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("clamp de maxParallel travou a fila")
	}
	if s, _ := statusByID(m, id2); s.State != StateQueued {
		t.Errorf("segundo job em %q; SetMaxParallel(0) deveria limitar a 1 slot", s.State)
	}

	close(release)
	waitState(t, m, id1, StateDone)
	waitState(t, m, id2, StateDone)
}

func TestWorkerRecordsError(t *testing.T) {
	m := NewManager()
	m.run = func(context.Context, string, Job) error {
		return errors.New("boom")
	}

	id, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	s := waitState(t, m, id, StateError)
	if s.Error != "boom" {
		t.Errorf("erro registrado = %q; queria %q", s.Error, "boom")
	}
	if s.Position != 0 {
		t.Errorf("tarefa com erro em position %d; queria 0", s.Position)
	}
}

func TestObserveEnrichesRunningTask(t *testing.T) {
	m := NewManager()
	m.run = func(_ context.Context, jobID string, _ Job) error {
		events.EmitProgress(jobID, events.KindCompress, "encoding · parte 1/3", 40)
		events.EmitProgress(jobID, events.KindCompress, "encoding · parte 2/3", 70)
		events.EmitDone(jobID, events.KindCompress, "/tmp/out.mp4", 1200, 1.2)
		return nil
	}
	events.SetEmitter(func(name string, data any) { m.Observe(name, data) })
	t.Cleanup(func() { events.SetEmitter(nil) })

	id, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	s := waitState(t, m, id, StateDone)

	if s.Percent != 100 || s.Stage != "done" {
		t.Errorf("ao concluir = {%%:%v stage:%q}; queria 100/done", s.Percent, s.Stage)
	}
	if s.PartsTotal != 3 || s.PartsDone != 1 {
		t.Errorf("partes = %d/%d; queria 1/3", s.PartsDone, s.PartsTotal)
	}
	if s.SizeBytes != 1200 {
		t.Errorf("SizeBytes = %d; queria 1200", s.SizeBytes)
	}
	if s.OutputPath != "/tmp/out.mp4" {
		t.Errorf("OutputPath = %q; queria /tmp/out.mp4", s.OutputPath)
	}

	events.EmitProgress(id, events.KindCompress, "encoding", 10)
	events.EmitError(id, events.KindCompress, "tarde demais")
	late, _ := statusByID(m, id)
	if late.State != StateDone || late.Percent != 100 || late.Error != "" {
		t.Errorf("evento tardio reescreveu a tarefa: %+v", late)
	}
}

func TestObserveRecordsErrorWhileRunning(t *testing.T) {
	proceed := make(chan struct{})
	m := NewManager()
	m.run = func(_ context.Context, jobID string, _ Job) error {
		events.EmitError(jobID, events.KindCompress, "falhou aqui")
		<-proceed
		return errors.New("falhou aqui")
	}
	events.SetEmitter(func(name string, data any) { m.Observe(name, data) })
	t.Cleanup(func() { events.SetEmitter(nil) })

	id, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})

	deadline := time.Now().Add(5 * time.Second)
	for {
		s, _ := statusByID(m, id)
		if s.Error == "falhou aqui" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("erro do evento não registrado: %+v", s)
		}
		time.Sleep(5 * time.Millisecond)
	}

	close(proceed)
	waitState(t, m, id, StateError)
}

func TestObserveIgnoresUnknownAndForeignEvents(t *testing.T) {
	m := NewManager()
	m.Observe("compress:progress", events.ProgressEvent{JobID: "nope", Kind: events.KindCompress, Percent: 50})
	m.Observe("compress:progress", map[string]any{"foo": "bar"})

	m.mu.Lock()
	m.tasks = append(m.tasks, &taskEntry{
		task:  Task{Kind: events.KindCompress, JobID: "j1"},
		state: StateRunning,
	})
	m.mu.Unlock()

	m.Observe("compress:progress", events.ProgressEvent{JobID: "j1", Kind: "audio", Percent: 10})
	if s, _ := statusByID(m, "j1"); s.Percent != 0 {
		t.Errorf("evento de outro kind alterou a tarefa: %+v", s)
	}

	m.Observe("compress:progress", events.ProgressEvent{JobID: "j1", Kind: events.KindCompress, Percent: 55})
	if s, _ := statusByID(m, "j1"); s.Percent != 55 {
		t.Errorf("evento próprio não enriqueceu a tarefa: %+v", s)
	}
}

func TestClearFinishedKeepsPending(t *testing.T) {
	release := make(chan struct{})
	started := make(chan string, 1)
	m := NewManager()
	m.run = func(_ context.Context, jobID string, _ Job) error {
		started <- jobID
		<-release
		return nil
	}

	id1, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	id2, _ := m.Enqueue(Task{Job: Job{InputPath: "b.mp4"}})
	id3, _ := m.Enqueue(Task{Job: Job{InputPath: "c.mp4"}})
	<-started
	m.Cancel(id2)

	m.ClearFinished()
	var ids []string
	for _, s := range m.List() {
		ids = append(ids, s.JobID)
	}
	if len(ids) != 2 || ids[0] != id1 || ids[1] != id3 {
		t.Errorf("após limpar = %v; queria [%s %s]", ids, id1, id3)
	}

	close(release)
	waitState(t, m, id1, StateDone)
	waitState(t, m, id3, StateDone)
	m.ClearFinished()
	if got := len(m.List()); got != 0 {
		t.Errorf("fila deveria ficar vazia, ficou com %d", got)
	}
}

func TestRemoveTask(t *testing.T) {
	m := NewManager()
	m.run = func(context.Context, string, Job) error { return nil }

	id, _ := m.Enqueue(Task{Job: Job{InputPath: "a.mp4"}})
	waitState(t, m, id, StateDone)

	m.Remove(id)
	if got := len(m.List()); got != 0 {
		t.Errorf("Remove deixou %d tarefas", got)
	}
	m.Remove("inexistente")
}

func TestQueueRunsTwoJobsToCompletion(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)

	m := NewManager()
	var mu sync.Mutex
	var names []string
	var dones []events.DoneEvent
	events.SetEmitter(func(name string, data any) {
		m.Observe(name, data)
		mu.Lock()
		defer mu.Unlock()
		names = append(names, name)
		if d, ok := data.(events.DoneEvent); ok {
			dones = append(dones, d)
		}
	})
	t.Cleanup(func() { events.SetEmitter(nil) })

	out1 := filepath.Join(t.TempDir(), "out1.mp4")
	out2 := filepath.Join(t.TempDir(), "out2.mp4")
	id1, _ := m.Enqueue(Task{Job: Job{InputPath: "in.mp4", OutputPath: out1, PresetID: "whatsapp-status"}})
	id2, _ := m.Enqueue(Task{Job: Job{InputPath: "in.mp4", OutputPath: out2, PresetID: "whatsapp-status"}})

	s1 := waitState(t, m, id1, StateDone)
	s2 := waitState(t, m, id2, StateDone)

	if s1.OutputPath != out1 || s1.SizeBytes <= 0 || s1.Percent != 100 {
		t.Errorf("status do job 1: %+v", s1)
	}
	if s2.OutputPath != out2 || s2.SizeBytes <= 0 || s2.Percent != 100 {
		t.Errorf("status do job 2: %+v", s2)
	}

	mu.Lock()
	defer mu.Unlock()
	counts := map[string]int{}
	for _, n := range names {
		counts[n]++
	}
	for _, n := range []string{"compress:queued", "compress:start", "compress:done"} {
		if counts[n] != 2 {
			t.Errorf("%s emitido %d vezes; queria 2 (todos: %v)", n, counts[n], names)
		}
	}
	if len(dones) != 2 || dones[0].OutputPath == dones[1].OutputPath {
		t.Errorf("dones = %+v; queria dois arquivos distintos", dones)
	}
}
