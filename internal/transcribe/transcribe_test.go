package transcribe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
)

func TestSpecValidate(t *testing.T) {
	base := Spec{InputPath: "in.mp4", OutputBase: "/tmp/out", Engine: "auto", Language: "auto", Format: "all"}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
	cases := map[string]Spec{
		"no input":       {OutputBase: "/tmp/out"},
		"no output":      {InputPath: "in.mp4"},
		"bad engine":     {InputPath: "in.mp4", OutputBase: "/tmp/o", Engine: "cloud"},
		"bad format":     {InputPath: "in.mp4", OutputBase: "/tmp/o", Format: "docx"},
		"bad language":   {InputPath: "in.mp4", OutputBase: "/tmp/o", Language: "de"},
		"negative start": {InputPath: "in.mp4", OutputBase: "/tmp/o", TrimStartSec: -1},
		"bad trim order": {InputPath: "in.mp4", OutputBase: "/tmp/o", TrimStartSec: 5, TrimEndSec: 2},
	}
	for name, spec := range cases {
		if err := spec.Validate(); !errors.Is(err, ErrInvalidSpec) {
			t.Errorf("%s: err = %v; want ErrInvalidSpec", name, err)
		}
	}
}

func TestSuggestName(t *testing.T) {
	got, err := SuggestName("/videos/meu video.mov", "")
	if err != nil || got != "meu video-transcript" {
		t.Errorf("SuggestName = %q, %v", got, err)
	}
	if got, _ := SuggestName("a.mp4", "-legenda"); got != "a-legenda" {
		t.Errorf("custom suffix = %q", got)
	}
	if _, err := SuggestName("", ""); !errors.Is(err, ErrInvalidSpec) {
		t.Errorf("empty input should fail, got %v", err)
	}
}

func TestPCMArgs(t *testing.T) {
	args := pcmArgs("in.mp4", "out.wav", 1.5, 3)
	joined := strings.Join(args, " ")
	for _, want := range []string{"-ss 1.500", "-to 3.000", "-i in.mp4", "-vn", "-ac 1", "-ar 16000", "-c:a pcm_s16le", "out.wav"} {
		if !strings.Contains(joined, want) {
			t.Errorf("pcmArgs missing %q in %q", want, joined)
		}
	}
	plain := pcmArgs("in.mp4", "out.wav", 0, 0)
	if strings.Contains(strings.Join(plain, " "), "-ss") || strings.Contains(strings.Join(plain, " "), "-to") {
		t.Errorf("no trim should omit -ss/-to: %v", plain)
	}
}

func TestWhisperArgs(t *testing.T) {
	args := whisperArgs("/tmp/base", Spec{ModelPath: "/models/ggml.bin", Language: "pt"}, "/tmp/w.wav")
	joined := strings.Join(args, " ")
	for _, want := range []string{"-m /models/ggml.bin", "-f /tmp/w.wav", "-oj", "-of /tmp/base", "-l pt"} {
		if !strings.Contains(joined, want) {
			t.Errorf("whisperArgs missing %q in %q", want, joined)
		}
	}
	if strings.Contains(joined, "-ml") {
		t.Errorf("word timestamps off should not add -ml: %q", joined)
	}
	words := whisperArgs("/tmp/base", Spec{ModelPath: "m", Language: "auto", WordTimestamps: true}, "w.wav")
	if !strings.Contains(strings.Join(words, " "), "-ml 1") {
		t.Errorf("word timestamps should add -ml 1: %v", words)
	}
	if !strings.Contains(strings.Join(words, " "), "-l auto") {
		t.Errorf("empty/auto language should map to auto: %v", words)
	}
}

func TestVoskArgs(t *testing.T) {
	args := voskArgs("/tmp/o.json", Spec{ModelPath: "/models/vosk", Language: "pt"}, "w.wav")
	joined := strings.Join(args, " ")
	for _, want := range []string{"-i w.wav", "-o /tmp/o.json", "-m /models/vosk", "-l pt"} {
		if !strings.Contains(joined, want) {
			t.Errorf("voskArgs missing %q in %q", want, joined)
		}
	}
	auto := strings.Join(voskArgs("/tmp/o.json", Spec{Language: "auto"}, "w.wav"), " ")
	if strings.Contains(auto, "-m") || strings.Contains(auto, "-l") {
		t.Errorf("auto language and no model should omit -m/-l: %q", auto)
	}
}

func TestWriteOutputs(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "video-transcript")
	segs := []Segment{{StartSec: 0, EndSec: 1, Text: "hello"}}

	files, text, size, err := writeOutputs(Spec{OutputBase: base, Format: "all"}, segs)
	if err != nil {
		t.Fatalf("writeOutputs: %v", err)
	}
	if len(files) != 3 || size == 0 || text != "hello" {
		t.Fatalf("all: files=%v size=%d text=%q", files, size, text)
	}
	for _, ext := range []string{"srt", "vtt", "txt"} {
		if _, err := os.Stat(base + "." + ext); err != nil {
			t.Errorf("missing %s: %v", ext, err)
		}
	}

	if files, _, _, err := writeOutputs(Spec{OutputBase: base, Format: "srt"}, segs); err != nil || len(files) != 1 {
		t.Errorf("srt only: files=%v err=%v", files, err)
	}
	if _, _, _, err := writeOutputs(Spec{OutputBase: base, Format: "docx"}, segs); !errors.Is(err, ErrInvalidSpec) {
		t.Errorf("unknown format should fail, got %v", err)
	}
}

// stubEngine installs an engine + lookPath so Transcribe can run without real
// tools, and restores everything on cleanup.
func stubEngine(t *testing.T) {
	t.Helper()
	withLookPath(t, fakeLookPath("whisper-cli"))

	oldProbe := probe
	probe = func(string) (*media.Info, error) {
		return &media.Info{HasAudio: true, DurationSec: 10}, nil
	}
	oldFF := ffmpegPath
	ffmpegPath = func() (string, error) { return "ffmpeg", nil }
	oldExtract := extractPCM
	extractPCM = func(_ context.Context, _, _, out string, _, _ float64) error {
		return os.WriteFile(out, []byte("wav"), 0o644)
	}
	oldEngine := runEngine
	runEngine = func(context.Context, string, Spec, string, string) ([]Segment, error) {
		return []Segment{{StartSec: 0, EndSec: 1, Text: "hello world"}}, nil
	}
	t.Cleanup(func() {
		probe = oldProbe
		ffmpegPath = oldFF
		extractPCM = oldExtract
		runEngine = oldEngine
	})
}

func TestTranscribeHappyPath(t *testing.T) {
	stubEngine(t)
	var done []events.DoneEvent
	events.SetEmitter(func(name string, data any) {
		if ev, ok := data.(events.DoneEvent); ok {
			done = append(done, ev)
		}
	})
	defer events.SetEmitter(nil)

	base := filepath.Join(t.TempDir(), "video-transcript")
	res, err := Transcribe(context.Background(), "job1", Spec{
		InputPath: "in.mp4", OutputBase: base, Engine: "auto", Language: "auto", Format: "all",
		ModelPath: "/models/ggml-small.bin",
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if len(res.Files) != 3 || res.Text != "hello world" {
		t.Fatalf("result = %+v", res)
	}
	if len(done) != 1 || done[0].Kind != events.KindTranscribe {
		t.Fatalf("done events = %+v", done)
	}
}

func TestTranscribeNoAudio(t *testing.T) {
	stubEngine(t)
	probe = func(string) (*media.Info, error) { return &media.Info{HasAudio: false}, nil }
	if _, err := Transcribe(context.Background(), "j", Spec{InputPath: "in.mp4", OutputBase: "/tmp/o", ModelPath: "/models/ggml.bin"}); err == nil || !strings.Contains(err.Error(), "no audio track") {
		t.Errorf("err = %v; want missing-audio error", err)
	}
}

func TestTranscribeNoEngine(t *testing.T) {
	withLookPath(t, fakeLookPath())
	_, err := Transcribe(context.Background(), "j", Spec{InputPath: "in.mp4", OutputBase: "/tmp/o", Engine: "auto"})
	if !errors.Is(err, ErrNoTranscriber) {
		t.Errorf("err = %v; want ErrNoTranscriber", err)
	}
}

func TestTranscribeNoModel(t *testing.T) {
	withLookPath(t, fakeLookPath("whisper-cli"))
	_, err := Transcribe(context.Background(), "j", Spec{InputPath: "in.mp4", OutputBase: "/tmp/o", Engine: "auto"})
	if !errors.Is(err, ErrNoModel) {
		t.Errorf("err = %v; want ErrNoModel", err)
	}
}

func TestTranscribeInvalidSpec(t *testing.T) {
	_, err := Transcribe(context.Background(), "j", Spec{})
	if !errors.Is(err, ErrInvalidSpec) {
		t.Errorf("err = %v; want ErrInvalidSpec", err)
	}
}

func TestTranscribeEngineError(t *testing.T) {
	stubEngine(t)
	runEngine = func(context.Context, string, Spec, string, string) ([]Segment, error) {
		return nil, errors.New("boom")
	}
	var errs []events.ErrorEvent
	events.SetEmitter(func(name string, data any) {
		if ev, ok := data.(events.ErrorEvent); ok {
			errs = append(errs, ev)
		}
	})
	defer events.SetEmitter(nil)

	_, err := Transcribe(context.Background(), "j", Spec{InputPath: "in.mp4", OutputBase: filepath.Join(t.TempDir(), "o"), Format: "all", ModelPath: "/models/ggml.bin"})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
	if len(errs) != 1 || errs[0].Kind != events.KindTranscribe {
		t.Errorf("error events = %+v", errs)
	}
}
