package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fabianoflorentino/vidctl/internal/compress"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/presets"
)

func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell bins não executam no Windows")
	}
}

func fakeBin(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestNewApp(t *testing.T) {
	if NewApp() == nil {
		t.Fatal("NewApp() == nil")
	}
}

func TestStartup(t *testing.T) {
	a := NewApp()
	a.startup(context.Background())
	if a.ctx == nil {
		t.Error("startup deve definir o contexto")
	}
	events.SetEmitter(nil)
}

func TestCheckFFmpegMissing(t *testing.T) {
	skipOnWindows(t)
	fakeBin(t, "outro", "#!/bin/sh\n")
	a := NewApp()
	st := a.CheckFFmpeg()
	if st.FFmpegOK {
		t.Error("FFmpegOK deveria ser false")
	}
	if !strings.Contains(st.Message, "ffmpeg, ffprobe") {
		t.Errorf("mensagem inesperada: %q", st.Message)
	}
}

func TestCheckFFmpegPartial(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	st := NewApp().CheckFFmpeg()
	if st.FFmpegOK {
		t.Error("FFmpegOK deveria ser false")
	}
	if !strings.Contains(st.Message, "ffprobe") || strings.Contains(st.Message, "ffprobe, ") {
		t.Errorf("mensagem inesperada: %q", st.Message)
	}
}

func TestCheckFFmpegOK(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if err := os.WriteFile(filepath.Join(dir, bin), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	st := NewApp().CheckFFmpeg()
	if !st.FFmpegOK || st.Message != "" {
		t.Errorf("esperava OK, got %+v", st)
	}
}

func TestJoinNames(t *testing.T) {
	if got := joinNames([]string{}); got != "" {
		t.Errorf("vazio: %q", got)
	}
	if got := joinNames([]string{"a"}); got != "a" {
		t.Errorf("único: %q", got)
	}
	if got := joinNames([]string{"a", "b", "c"}); got != "a, b, c" {
		t.Errorf("múltiplos: %q", got)
	}
}

func TestGetPresets(t *testing.T) {
	got := NewApp().GetPresets()
	if len(got) != len(presets.List()) {
		t.Errorf("got %d presets, want %d", len(got), len(presets.List()))
	}
}

func TestCompressEmptyInput(t *testing.T) {
	if _, err := NewApp().Compress(compress.Job{PresetID: "youtube"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCompressEmptyPreset(t *testing.T) {
	if _, err := NewApp().Compress(compress.Job{InputPath: "in.mp4"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMediaInfoError(t *testing.T) {
	skipOnWindows(t)
	fakeBin(t, "ffprobe", "#!/bin/sh\nexit 1\n")
	if _, err := NewApp().GetMediaInfo("/tmp/x.mp4"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMediaInfoSuccess(t *testing.T) {
	skipOnWindows(t)
	in := filepath.Join(t.TempDir(), "input.mp4")
	if err := os.WriteFile(in, make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := `#!/bin/sh
printf '%s\n' '{"streams":[{"codec_type":"video","codec_name":"h264","width":640,"height":480,"duration":5.0}],"format":{"duration":5.0}}'
`
	if err := os.WriteFile(filepath.Join(dir, "ffprobe"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	info, err := NewApp().GetMediaInfo(in)
	if err != nil {
		t.Fatalf("GetMediaInfo failed: %v", err)
	}
	if info.Width != 640 || !info.HasVideo || info.DurationSec != 5.0 {
		t.Errorf("info inesperada: %+v", info)
	}
}

func TestCancelUnknownJob(t *testing.T) {
	if err := NewApp().Cancel("nope"); err != nil {
		t.Fatalf("Cancel erro: %v", err)
	}
}

func TestVideoFilterPattern(t *testing.T) {
	for _, ext := range []string{"mp4", "mkv", "mov", "avi", "webm", "m4v", "ts", "flv"} {
		if !strings.Contains(videoFilterPattern, "*."+ext) {
			t.Errorf("padrão deve contemplar *.%s: %q", ext, videoFilterPattern)
		}
	}
}

func TestCancelRegisteredJob(t *testing.T) {
	a := NewApp()
	canceled := false
	a.jobs.Register("j1", func() { canceled = true })
	if err := a.Cancel("j1"); err != nil {
		t.Fatalf("Cancel erro: %v", err)
	}
	if !canceled {
		t.Error("cancel function should have been called")
	}
}

func TestOpenFolderMissingOpener(t *testing.T) {
	skipOnWindows(t)
	fakeBin(t, "algo", "#!/bin/sh\n")
	if err := NewApp().OpenFolder("/tmp/algum/arquivo.mp4"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCompressSuccess(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg": `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
case "$out" in /dev/null) ;; *) printf 'hi' > "$out" ;; esac
echo out_time_us=1000000
exit 0
`,
		"ffprobe": `#!/bin/sh
printf '%s\n' '{"streams":[
	{"codec_type":"video","codec_name":"h264","width":320,"height":240,"duration":2.0},
	{"codec_type":"audio","codec_name":"aac"}
],"format":{"duration":2.0}}'
`,
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	finished := make(chan struct{})
	events.SetEmitter(func(name string, _ any) {
		if name == "compress:done" || name == "compress:error" {
			select {
			case finished <- struct{}{}:
			default:
			}
		}
	})
	defer events.SetEmitter(nil)

	a := NewApp()
	output := filepath.Join(t.TempDir(), "out.mp4")
	jobID, err := a.Compress(compress.Job{InputPath: "in.mp4", OutputPath: output, PresetID: "whatsapp-status"})
	if err != nil || jobID == "" {
		t.Fatalf("Compress: jobID=%q err=%v", jobID, err)
	}

	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatalf("job %q não terminou", jobID)
	}

	if st, err := os.Stat(output); err != nil || st.Size() == 0 {
		t.Errorf("arquivo de saída ausente: err=%v", err)
	}
}
