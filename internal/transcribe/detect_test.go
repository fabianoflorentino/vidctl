package transcribe

import (
	"errors"
	"testing"
)

// fakeLookPath returns a lookPath that reports exactly the given binaries as
// present.
func fakeLookPath(present ...string) func(string) (string, error) {
	set := make(map[string]bool, len(present))
	for _, p := range present {
		set[p] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
}

func withLookPath(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	old := lookPath
	lookPath = fn
	ResetDetection()
	t.Cleanup(func() {
		lookPath = old
		ResetDetection()
	})
}

func TestSupportedEngines(t *testing.T) {
	got := SupportedEngines()
	if len(got) != 2 || got[0] != EngineWhisper || got[1] != EngineVosk {
		t.Errorf("SupportedEngines = %v", got)
	}
}

func TestAvailableEnginesDetectsAvailability(t *testing.T) {
	withLookPath(t, fakeLookPath("vosk-transcriber"))
	if got := AvailableEngines(); len(got) != 1 || got[0] != EngineVosk {
		t.Errorf("AvailableEngines = %v; want [vosk]", got)
	}
}

func TestAvailableEnginesPrefersWhisper(t *testing.T) {
	withLookPath(t, fakeLookPath("whisper-cli", "vosk-transcriber"))
	got := AvailableEngines()
	if len(got) != 2 || got[0] != EngineWhisper {
		t.Errorf("AvailableEngines = %v; want [whispercpp vosk]", got)
	}
}

func TestAvailableEnginesNone(t *testing.T) {
	withLookPath(t, fakeLookPath())
	if got := AvailableEngines(); len(got) != 0 {
		t.Errorf("AvailableEngines = %v; want empty", got)
	}
}

func TestAvailableEnginesIsCached(t *testing.T) {
	withLookPath(t, fakeLookPath("whisper-cli"))
	if got := AvailableEngines(); len(got) != 1 {
		t.Fatalf("first call = %v", got)
	}
	// Changing the PATH without resetting the cache must not be seen.
	lookPath = fakeLookPath("vosk-transcriber")
	if got := AvailableEngines(); len(got) != 1 || got[0] != EngineWhisper {
		t.Errorf("cached result changed: %v", got)
	}
	ResetDetection()
	if got := AvailableEngines(); len(got) != 1 || got[0] != EngineVosk {
		t.Errorf("after reset = %v; want [vosk]", got)
	}
}

func TestPickEngine(t *testing.T) {
	withLookPath(t, fakeLookPath("whisper-cli", "vosk-transcriber"))

	if got, err := pickEngine(""); err != nil || got != EngineWhisper {
		t.Errorf("pickEngine(\"\") = %q, %v", got, err)
	}
	if got, err := pickEngine(EngineAuto); err != nil || got != EngineWhisper {
		t.Errorf("pickEngine(auto) = %q, %v", got, err)
	}
	if got, err := pickEngine(EngineVosk); err != nil || got != EngineVosk {
		t.Errorf("pickEngine(vosk) = %q, %v", got, err)
	}
	if _, err := pickEngine("nope"); !errors.Is(err, ErrNoTranscriber) {
		t.Errorf("unknown engine should be ErrNoTranscriber, got %v", err)
	}
}

func TestPickEngineNotInstalled(t *testing.T) {
	withLookPath(t, fakeLookPath())
	if _, err := pickEngine(EngineAuto); !errors.Is(err, ErrNoTranscriber) {
		t.Errorf("auto without engines should be ErrNoTranscriber, got %v", err)
	}
	if _, err := pickEngine(EngineWhisper); !errors.Is(err, ErrNoTranscriber) {
		t.Errorf("missing whisper should be ErrNoTranscriber, got %v", err)
	}
}
