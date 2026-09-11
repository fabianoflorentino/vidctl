package events

import "testing"

func capture(t *testing.T) (map[string][]any, func()) {
	t.Helper()
	got := map[string][]any{}
	SetEmitter(func(name string, data any) {
		got[name] = append(got[name], data)
	})
	return got, func() { SetEmitter(nil) }
}

func TestEmitProgress(t *testing.T) {
	got, restore := capture(t)
	defer restore()

	EmitProgress("job-1", "pass1/2", 42.5)

	evts := got["compress:progress"]
	if len(evts) != 1 {
		t.Fatalf("compress:progress = %d eventos; want 1", len(evts))
	}
	p, ok := evts[0].(ProgressEvent)
	if !ok {
		t.Fatalf("payload não é ProgressEvent: %T", evts[0])
	}
	if p.JobID != "job-1" || p.Stage != "pass1/2" || p.Percent != 42.5 {
		t.Errorf("ProgressEvent inesperado: %+v", p)
	}
}

func TestEmitDone(t *testing.T) {
	got, restore := capture(t)
	defer restore()

	EmitDone("job-2", "/tmp/out.mp4", 1024, 0.001)

	evts := got["compress:done"]
	if len(evts) != 1 {
		t.Fatalf("compress:done = %d eventos; want 1", len(evts))
	}
	d, ok := evts[0].(DoneEvent)
	if !ok {
		t.Fatalf("payload não é DoneEvent: %T", evts[0])
	}
	if d.JobID != "job-2" || d.OutputPath != "/tmp/out.mp4" || d.SizeBytes != 1024 || d.SizeMB != 0.001 {
		t.Errorf("DoneEvent inesperado: %+v", d)
	}
}

func TestEmitError(t *testing.T) {
	got, restore := capture(t)
	defer restore()

	EmitError("job-3", "deu ruim")

	evts := got["compress:error"]
	if len(evts) != 1 {
		t.Fatalf("compress:error = %d eventos; want 1", len(evts))
	}
	e, ok := evts[0].(ErrorEvent)
	if !ok {
		t.Fatalf("payload não é ErrorEvent: %T", evts[0])
	}
	if e.JobID != "job-3" || e.Error != "deu ruim" {
		t.Errorf("ErrorEvent inesperado: %+v", e)
	}
}

func TestEmitWithoutEmitterIsNoop(t *testing.T) {
	SetEmitter(nil)
	emit("compress:progress", ProgressEvent{})
	EmitProgress("job-4", "encoding", 10)
	EmitDone("job-4", "/tmp/x.mp4", 1, 0.1)
	EmitError("job-4", "ignorado")
}
