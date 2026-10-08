// Package events centralizes the events emitted by the backend to the frontend.
package events

import "sync"

// KindCompress identifies compression payloads; audio and transcription join
// the queue in later phases with their own kinds.
const KindCompress = "compress"

// Emitter is a function that dispatches a named event with a payload.
type Emitter func(name string, data any)

var (
	mu      sync.RWMutex
	current Emitter
)

// SetEmitter registers the emitter function. Safe to call concurrently with
// emits: queue workers may still be draining when the app is torn down.
func SetEmitter(fn Emitter) {
	mu.Lock()
	current = fn
	mu.Unlock()
}

func emit(name string, data any) {
	mu.RLock()
	fn := current
	mu.RUnlock()
	if fn != nil {
		fn(name, data)
	}
}

// ProgressEvent reports compression progress for a job.
type ProgressEvent struct {
	JobID   string  `json:"jobId"`
	Kind    string  `json:"kind"`
	Stage   string  `json:"stage"` // "pass1/2" | "pass2/2" | "encoding" | "done"
	Percent float64 `json:"percent"`
}

// DoneEvent reports a successfully finished job.
type DoneEvent struct {
	JobID      string  `json:"jobId"`
	Kind       string  `json:"kind"`
	OutputPath string  `json:"outputPath"`
	SizeMB     float64 `json:"sizeMB"`
	SizeBytes  int64   `json:"sizeBytes"`
}

// ErrorEvent reports a failed job.
type ErrorEvent struct {
	JobID string `json:"jobId"`
	Kind  string `json:"kind"`
	Error string `json:"error"`
}

// QueuedEvent reports a job waiting in the queue.
type QueuedEvent struct {
	JobID    string `json:"jobId"`
	Position int    `json:"position"`
}

// StartEvent reports a queued job that began executing.
type StartEvent struct {
	JobID string `json:"jobId"`
}

// EmitProgress emits a progress update.
func EmitProgress(jobID, kind, stage string, percent float64) {
	emit("compress:progress", ProgressEvent{JobID: jobID, Kind: kind, Stage: stage, Percent: percent})
}

// EmitDone emits a completion event.
func EmitDone(jobID, kind, outputPath string, sizeBytes int64, sizeMB float64) {
	emit("compress:done", DoneEvent{JobID: jobID, Kind: kind, OutputPath: outputPath, SizeBytes: sizeBytes, SizeMB: sizeMB})
}

// EmitError emits a failure event.
func EmitError(jobID, kind, msg string) {
	emit("compress:error", ErrorEvent{JobID: jobID, Kind: kind, Error: msg})
}

// EmitQueued emits that a job entered the waiting line.
func EmitQueued(jobID string, position int) {
	emit("compress:queued", QueuedEvent{JobID: jobID, Position: position})
}

// EmitStart emits that a queued job began running.
func EmitStart(jobID string) {
	emit("compress:start", StartEvent{JobID: jobID})
}
