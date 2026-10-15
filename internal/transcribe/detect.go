// Package transcribe generates subtitles and plain text from the audio of a
// video using local, private speech-to-text engines (whisper.cpp or vosk).
package transcribe

import (
	"os/exec"
	"sync"
)

// Engine identifiers accepted by Spec.Engine.
const (
	EngineAuto    = "auto"
	EngineWhisper = "whispercpp"
	EngineVosk    = "vosk"
)

// engineBins maps an engine to the CLI names it may be installed as, in
// preference order.
var engineBins = map[string][]string{
	EngineWhisper: {"whisper-cli", "whisper-cpp", "whisper"},
	EngineVosk:    {"vosk-transcriber", "vosk-transcriber-cli"},
}

// engineOrder is the preference order used by "auto".
var engineOrder = []string{EngineWhisper, EngineVosk}

// lookPath is indirected so tests can fake an installation.
var lookPath = exec.LookPath

var (
	detectMu   sync.Mutex
	detected   []string
	detectedOK bool
)

// SupportedEngines returns every engine id in preference order.
func SupportedEngines() []string {
	return append([]string(nil), engineOrder...)
}

// AvailableEngines reports which engines are installed, cached for the session.
// The list follows the preference order.
func AvailableEngines() []string {
	detectMu.Lock()
	defer detectMu.Unlock()
	if detectedOK {
		return append([]string(nil), detected...)
	}
	var out []string
	for _, e := range engineOrder {
		if _, ok := resolveEngineLocked(e); ok {
			out = append(out, e)
		}
	}
	detected = out
	detectedOK = true
	return append([]string(nil), out...)
}

// ResetDetection drops the cached engine detection. The next AvailableEngines
// call probes the PATH again.
func ResetDetection() {
	detectMu.Lock()
	detected = nil
	detectedOK = false
	detectMu.Unlock()
}

// resolveEngine returns the path of the binary backing an engine, or false when
// the engine is unknown or not installed.
func resolveEngine(name string) (string, bool) {
	detectMu.Lock()
	defer detectMu.Unlock()
	return resolveEngineLocked(name)
}

// resolveEngineLocked is resolveEngine without taking the lock; callers must
// already hold detectMu.
func resolveEngineLocked(name string) (string, bool) {
	bins, ok := engineBins[name]
	if !ok {
		return "", false
	}
	for _, b := range bins {
		if p, err := lookPath(b); err == nil {
			return p, true
		}
	}
	return "", false
}

// pickEngine turns a requested engine into the concrete, installed one. "auto"
// (or empty) picks the first available engine.
func pickEngine(requested string) (string, error) {
	if requested == "" {
		requested = EngineAuto
	}
	if requested == EngineAuto {
		if list := AvailableEngines(); len(list) > 0 {
			return list[0], nil
		}
		return "", ErrNoTranscriber
	}
	if _, ok := resolveEngine(requested); !ok {
		return "", ErrNoTranscriber
	}
	return requested, nil
}
