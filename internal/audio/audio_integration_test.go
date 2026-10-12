package audio

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/events"
)

func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell bins do not run on Windows")
	}
}

// installFakeTools installs scripted ffmpeg/ffprobe in a temporary PATH. The
// ffmpeg script writes the last argument (the output) so Extract reports a
// real size, echoes progress lines, and records its argv in the returned log
// path for inspection.
func installFakeTools(t *testing.T, withAudio bool) string {
	t.Helper()
	dir := t.TempDir()

	ffmpeg := `#!/bin/sh
if [ -n "$FAKE_ARGV_LOG" ]; then
	printf '%s\n' "$*" > "$FAKE_ARGV_LOG"
fi
out=""
for a in "$@"; do out="$a"; done
mkdir -p "$(dirname "$out")"
printf 'fake audio bytes' > "$out"
echo out_time_us=500000
echo progress=continue
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte(ffmpeg), 0o755); err != nil {
		t.Fatal(err)
	}

	videoStream := `{"codec_type":"video","codec_name":"h264","width":640,"height":480}`
	streams := videoStream
	if withAudio {
		streams += `,{"codec_type":"audio","codec_name":"aac"}`
	}
	ffprobe := "#!/bin/sh\nprintf '%s\\n' '{\"streams\":[" + streams + "],\"format\":{\"duration\":5.0}}'\n"
	if err := os.WriteFile(filepath.Join(dir, "ffprobe"), []byte(ffprobe), 0o755); err != nil {
		t.Fatal(err)
	}

	log := filepath.Join(t.TempDir(), "argv.txt")
	t.Setenv("FAKE_ARGV_LOG", log)
	t.Setenv("PATH", dir)
	return log
}

func readArgv(t *testing.T, log string) []string {
	t.Helper()
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("argv not captured: %v", err)
	}
	var out []string
	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == ' ' {
			out = append(out, string(data[start:i]))
			start = i + 1
		}
	}
	return out
}

func TestExtractSuccess(t *testing.T) {
	skipOnWindows(t)
	log := installFakeTools(t, true)

	var dones []events.DoneEvent
	events.SetEmitter(func(name string, data any) {
		if d, ok := data.(events.DoneEvent); ok {
			dones = append(dones, d)
		}
	})
	defer events.SetEmitter(nil)

	input := "/videos/input.mp4"
	out := filepath.Join(t.TempDir(), "out", "input-audio.mp3")
	spec := Spec{InputPath: input, Codec: "mp3", BitrateKbps: 192, OutputPath: out}

	size, err := Extract(context.Background(), "job-audio", input, spec)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if size <= 0 {
		t.Errorf("size = %d, esperava > 0", size)
	}
	if st, err := os.Stat(out); err != nil || st.Size() == 0 {
		t.Errorf("output file missing: err=%v", err)
	}
	if len(dones) != 1 || dones[0].Kind != events.KindAudio || dones[0].OutputPath != out {
		t.Errorf("dones = %+v; want one audio DoneEvent for %q", dones, out)
	}

	argv := readArgv(t, log)
	got := ""
	progress := false
	for i, a := range argv {
		if a == "-i" && i+1 < len(argv) {
			got = argv[i+1]
		}
		if a == "-progress" && i+1 < len(argv) && argv[i+1] == "pipe:1" {
			progress = true
		}
	}
	if got != input {
		t.Errorf("entrada no argv = %q, want %q", got, input)
	}
	if !progress {
		t.Errorf("argv %v does not enable progress via -progress pipe:1", argv)
	}
}

func TestExtractEmitsProgress(t *testing.T) {
	skipOnWindows(t)
	installFakeTools(t, true)

	var progresses []events.ProgressEvent
	events.SetEmitter(func(name string, data any) {
		if p, ok := data.(events.ProgressEvent); ok {
			progresses = append(progresses, p)
		}
	})
	defer events.SetEmitter(nil)

	out := filepath.Join(t.TempDir(), "in-audio.mp3")
	_, err := Extract(context.Background(), "job-prog", "/videos/in.mp4",
		Spec{InputPath: "/videos/in.mp4", Codec: "mp3", OutputPath: out})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(progresses) == 0 {
		t.Fatal("esperava eventos de progresso")
	}
	last := progresses[len(progresses)-1]
	if last.Kind != events.KindAudio || last.Stage != "extracting" {
		t.Errorf("last progress = %+v", last)
	}
}

func TestExtractVideoWithoutAudio(t *testing.T) {
	skipOnWindows(t)
	installFakeTools(t, false)

	var errors []events.ErrorEvent
	events.SetEmitter(func(name string, data any) {
		if e, ok := data.(events.ErrorEvent); ok {
			errors = append(errors, e)
		}
	})
	defer events.SetEmitter(nil)

	out := filepath.Join(t.TempDir(), "in-audio.mp3")
	_, err := Extract(context.Background(), "job-noaud", "/videos/in.mp4",
		Spec{InputPath: "/videos/in.mp4", Codec: "mp3", OutputPath: out})
	if err == nil {
		t.Fatal("expected an error for a video without audio")
	}
	if len(errors) != 1 {
		t.Errorf("erro emitidos = %d, queria 1", len(errors))
	}
}

func TestExtractInvalidSpecFailsBeforeRunning(t *testing.T) {
	skipOnWindows(t)
	installFakeTools(t, true)

	if _, err := Extract(context.Background(), "job-inv", "/videos/in.mp4",
		Spec{InputPath: "/videos/in.mp4", Codec: "ogg", OutputPath: "/tmp/o.mp3"}); err == nil {
		t.Fatal("esperava erro de formato desconhecido")
	}
}

func TestExtractCanceledContext(t *testing.T) {
	skipOnWindows(t)
	installFakeTools(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out := filepath.Join(t.TempDir(), "in-audio.mp3")
	if _, err := Extract(ctx, "job-cancel", "/videos/in.mp4",
		Spec{InputPath: "/videos/in.mp4", Codec: "mp3", OutputPath: out}); err == nil {
		t.Fatal("esperava erro de contexto cancelado")
	}
}
