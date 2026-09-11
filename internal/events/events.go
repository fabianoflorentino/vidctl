// Package events centralizes the events emitted by the backend to the frontend.
package events

// Emitter is a function that dispatches a named event with a payload.
type Emitter func(name string, data any)

var current Emitter

// SetEmitter registers the emitter function. Called once during app startup.
func SetEmitter(fn Emitter) {
	current = fn
}

func emit(name string, data any) {
	if current != nil {
		current(name, data)
	}
}

// ProgressEvent reports compression progress for a job.
type ProgressEvent struct {
	JobID   string  `json:"jobId"`
	Stage   string  `json:"stage"` // "pass1/2" | "pass2/2" | "encoding" | "done"
	Percent float64 `json:"percent"`
}

// DoneEvent reports a successfully finished job.
type DoneEvent struct {
	JobID      string  `json:"jobId"`
	OutputPath string  `json:"outputPath"`
	SizeMB     float64 `json:"sizeMB"`
	SizeBytes  int64   `json:"sizeBytes"`
}

// ErrorEvent reports a failed job.
type ErrorEvent struct {
	JobID string `json:"jobId"`
	Error string `json:"error"`
}

// EmitProgress emits a progress update.
func EmitProgress(jobID, stage string, percent float64) {
	emit("compress:progress", ProgressEvent{JobID: jobID, Stage: stage, Percent: percent})
}

// EmitDone emits a completion event.
func EmitDone(jobID, outputPath string, sizeBytes int64, sizeMB float64) {
	emit("compress:done", DoneEvent{JobID: jobID, OutputPath: outputPath, SizeBytes: sizeBytes, SizeMB: sizeMB})
}

// EmitError emits a failure event.
func EmitError(jobID, msg string) {
	emit("compress:error", ErrorEvent{JobID: jobID, Error: msg})
}
