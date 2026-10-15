package app

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fabianoflorentino/vidctl/internal/audio"
	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/compress"
	"github.com/fabianoflorentino/vidctl/internal/config"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/notify"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/transcribe"
	"github.com/fabianoflorentino/vidctl/internal/update"
)

func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell bins do not run on Windows")
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

func TestNewAppWithoutConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("APPDATA", "")

	// Without a configuration folder the app must still open with the defaults.
	a := NewApp()
	if a == nil {
		t.Fatal("NewApp() == nil")
	}
	if a.store != nil {
		t.Error("NewApp should end up without a store when there is no configuration folder")
	}
	if got := a.GetConfig(); got != config.Defaults() {
		t.Errorf("GetConfig() = %+v, want defaults", got)
	}
}

// newTestApp builds an App whose settings live in a temporary directory, and
// guarantees the process-wide binary overrides are cleared afterwards.
func newTestApp(t *testing.T) *App {
	t.Helper()
	store, err := config.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("config.NewStore: %v", err)
	}
	t.Cleanup(cmdutil.ClearOverrides)
	return newApp(store)
}

func TestStartup(t *testing.T) {
	a := newTestApp(t)
	a.startup(context.Background())
	if a.ctx == nil {
		t.Error("startup deve definir o contexto")
	}
	events.SetEmitter(nil)
}

func TestStartupAppliesConfiguredToolPaths(t *testing.T) {
	binDir := t.TempDir()
	custom := filepath.Join(binDir, "ffmpeg")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t)
	cfg := config.Defaults()
	cfg.FFmpegPath = custom
	cfg.FFprobePath = filepath.Join(binDir, "ffprobe")
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Um startup limpo precisa reidratar os overrides a partir do arquivo.
	cmdutil.ClearOverrides()
	a2 := newTestApp(t)
	if err := a2.store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	a2.startup(context.Background())
	events.SetEmitter(nil)

	got, ok := cmdutil.Override("ffmpeg")
	if !ok || got != custom {
		t.Errorf("override de ffmpeg = %q (ok=%v), want %q", got, ok, custom)
	}
}

func TestStartupWithoutStoreKeepsDefaults(t *testing.T) {
	t.Cleanup(cmdutil.ClearOverrides)
	a := newApp(nil)
	a.startup(context.Background())
	events.SetEmitter(nil)

	if _, ok := cmdutil.Override("ffmpeg"); ok {
		t.Error("startup without a store should not configure an override")
	}
}

func TestGetConfigDefaults(t *testing.T) {
	got := newTestApp(t).GetConfig()
	if got != config.Defaults() {
		t.Errorf("GetConfig() = %+v, want defaults", got)
	}
}

func TestGetConfigWithoutStore(t *testing.T) {
	got := newApp(nil).GetConfig()
	if got != config.Defaults() {
		t.Errorf("GetConfig() = %+v, want defaults", got)
	}
}

func TestSaveConfigRoundtrip(t *testing.T) {
	a := newTestApp(t)
	want := config.Defaults()
	want.PresetID = "youtube"
	want.SizeMB = 42
	want.CRF = 20
	want.OutputDir = t.TempDir()
	want.Language = "en"
	want.MaxParallel = 2
	want.NotifyOnDone = true
	want.OpenFolderOnDone = true

	if err := a.SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	got := a.GetConfig()
	if got != want {
		t.Errorf("roundtrip = %+v, want %+v", got, want)
	}
}

func TestSaveConfigAppliesToolPathsImmediately(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t)
	cfg := config.Defaults()
	cfg.FFmpegPath = custom
	cfg.FFprobePath = custom
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		got, ok := cmdutil.Override(bin)
		if !ok || got != custom {
			t.Errorf("override de %s = %q (ok=%v), want %q", bin, got, ok, custom)
		}
	}
}

func TestSaveConfigClearsToolPathsWhenBlank(t *testing.T) {
	a := newTestApp(t)
	custom := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.FFmpegPath = custom
	cfg.FFprobePath = custom
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("primeiro SaveConfig: %v", err)
	}

	blank := config.Defaults()
	if err := a.SaveConfig(blank); err != nil {
		t.Fatalf("segundo SaveConfig: %v", err)
	}
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, ok := cmdutil.Override(bin); ok {
			t.Errorf("override de %s should ter sido removido", bin)
		}
	}
}

func TestSaveConfigRejectsInvalidAndKeepsOverride(t *testing.T) {
	a := newTestApp(t)
	custom := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	valid := config.Defaults()
	valid.FFmpegPath = custom
	if err := a.SaveConfig(valid); err != nil {
		t.Fatalf("valid SaveConfig: %v", err)
	}

	invalid := valid
	invalid.CRF = 99
	if err := a.SaveConfig(invalid); err == nil {
		t.Fatal("SaveConfig should rejeitar crf fora da faixa")
	}
	if got, ok := cmdutil.Override("ffmpeg"); !ok || got != custom {
		t.Errorf("override = %q (ok=%v), want o anterior %q", got, ok, custom)
	}
	got := a.GetConfig()
	if got.CRF != valid.CRF {
		t.Errorf("config salvo tem crf %v, want %v intacto", got.CRF, valid.CRF)
	}
}

func TestSaveConfigWithoutStore(t *testing.T) {
	err := newApp(nil).SaveConfig(config.Defaults())
	if err == nil {
		t.Fatal("SaveConfig sem store should falhar")
	}
	if !strings.Contains(err.Error(), "configuration") {
		t.Errorf("mensagem inesperada: %q", err)
	}
}

func TestCheckFFmpegUsesConfiguredPath(t *testing.T) {
	skipOnWindows(t)
	// The PATH has neither ffmpeg nor ffprobe; only the configured paths work.
	t.Setenv("PATH", t.TempDir())
	binDir := t.TempDir()
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if err := os.WriteFile(filepath.Join(binDir, bin), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	a := newTestApp(t)
	cfg := config.Defaults()
	cfg.FFmpegPath = filepath.Join(binDir, "ffmpeg")
	cfg.FFprobePath = filepath.Join(binDir, "ffprobe")
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	st := a.CheckFFmpeg()
	if !st.FFmpegOK {
		t.Errorf("expected OK com os caminhos configurados, got %+v", st)
	}
}

func TestCheckFFmpegFailsWithBrokenConfiguredPath(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if err := os.WriteFile(filepath.Join(dir, bin), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	a := newTestApp(t)
	cfg := config.Defaults()
	cfg.FFmpegPath = filepath.Join(t.TempDir(), "ffmpeg-que-nao-existe")
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	st := a.CheckFFmpeg()
	if st.FFmpegOK {
		t.Error("FFmpegOK should be false with an invalid configured path")
	}
	if !strings.Contains(st.Message, "ffmpeg") {
		t.Errorf("mensagem should citar o ffmpeg: %q", st.Message)
	}
}

func TestCheckFFmpegMissing(t *testing.T) {
	skipOnWindows(t)
	fakeBin(t, "outro", "#!/bin/sh\n")
	a := NewApp()
	st := a.CheckFFmpeg()
	if st.FFmpegOK {
		t.Error("FFmpegOK should ser false")
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
		t.Error("FFmpegOK should ser false")
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
		t.Errorf("expected OK, got %+v", st)
	}
}

func TestJoinNames(t *testing.T) {
	if got := joinNames([]string{}); got != "" {
		t.Errorf("empty: %q", got)
	}
	if got := joinNames([]string{"a"}); got != "a" {
		t.Errorf("single: %q", got)
	}
	if got := joinNames([]string{"a", "b", "c"}); got != "a, b, c" {
		t.Errorf("multiple: %q", got)
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

func TestGetThumbnailError(t *testing.T) {
	if _, err := NewApp().GetThumbnail(filepath.Join(t.TempDir(), "nope.mp4"), 2); err == nil {
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

func TestDebugEnabled(t *testing.T) {
	app := NewApp()
	t.Setenv("VIDCTL_DEBUG", "1")
	if !app.DebugEnabled() {
		t.Fatal("DebugEnabled should ser true com VIDCTL_DEBUG=1")
	}
	t.Setenv("VIDCTL_DEBUG", "")
	if app.DebugEnabled() {
		t.Fatal("DebugEnabled should ser false com VIDCTL_DEBUG empty")
	}
}

func TestDebugLogSemSaidaQuandoDesabilitado(t *testing.T) {
	t.Setenv("VIDCTL_DEBUG", "")
	NewApp().DebugLog("linha silenciosa")
}

func TestVideoFilterPattern(t *testing.T) {
	for _, ext := range []string{"mp4", "mkv", "mov", "avi", "webm", "m4v", "ts", "flv"} {
		if !strings.Contains(videoFilterPattern, "*."+ext) {
			t.Errorf("default deve contemplar *.%s: %q", ext, videoFilterPattern)
		}
	}
}

func waitTaskTerminal(t *testing.T, a *App, jobID string) compress.TaskStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last compress.TaskStatus
	for time.Now().Before(deadline) {
		for _, s := range a.GetTasks() {
			if s.JobID != jobID {
				continue
			}
			last = s
			if s.State != compress.StateQueued && s.State != compress.StateRunning {
				return s
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("job %s did not finish, stayed in %q (%+v)", jobID, last.State, last)
	return last
}

func TestCancelRunningJob(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg": "#!/bin/sh\nsleep 3\nexit 0\n",
		"ffprobe": `#!/bin/sh
printf '%s\n' '{"streams":[{"codec_type":"video","codec_name":"h264","width":320,"height":240,"duration":2.0}],"format":{"duration":2.0}}'
`,
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	a := NewApp()
	jobID, err := a.Compress(compress.Job{
		InputPath:  "in.mp4",
		OutputPath: filepath.Join(t.TempDir(), "out.mp4"),
		PresetID:   "whatsapp-status",
	})
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}

	if err := a.Cancel(jobID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if s := waitTaskTerminal(t, a, jobID); s.State != compress.StateCanceled {
		t.Errorf("estado final = %q; want canceled", s.State)
	}
}

func TestCompressMultiple(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg": "#!/bin/sh\nsleep 3\nexit 0\n",
		"ffprobe": `#!/bin/sh
printf '%s\n' '{"streams":[{"codec_type":"video","codec_name":"h264","width":320,"height":240,"duration":2.0}],"format":{"duration":2.0}}'
`,
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	a := NewApp()
	acks := a.CompressMultiple([]compress.Job{
		{InputPath: "", PresetID: "whatsapp-status"},
		{InputPath: "a.mp4", PresetID: ""},
		{InputPath: "a.mp4", PresetID: "whatsapp-status"},
		{InputPath: "b.mp4", PresetID: "whatsapp-status"},
	})
	if len(acks) != 4 {
		t.Fatalf("acks = %d; want 4", len(acks))
	}
	if acks[0].JobID != "" || !strings.Contains(acks[0].Error, "no video") {
		t.Errorf("ack[0] = %+v; want a video error", acks[0])
	}
	if acks[1].JobID != "" || !strings.Contains(acks[1].Error, "no preset") {
		t.Errorf("ack[1] = %+v; want erro de preset", acks[1])
	}
	// Both valid jobs must be enqueued and present in the queue. Positions are
	// not asserted: they depend on worker timing and a worker that finishes (or
	// errors) before the next enqueue makes positions shift under -race.
	if acks[2].JobID == "" || acks[3].JobID == "" || acks[2].JobID == acks[3].JobID {
		t.Errorf("acks[2..3] without unique jobId: %+v %+v", acks[2], acks[3])
	}
	if tasks := a.GetTasks(); len(tasks) != 2 {
		t.Errorf("GetTasks = %d tasks; want 2", len(tasks))
	}

	for _, id := range []string{acks[2].JobID, acks[3].JobID} {
		a.Cancel(id)
		waitTaskTerminal(t, a, id)
	}
	if left := a.ClearFinished(); len(left) != 0 {
		t.Errorf("ClearFinished left %d tasks; want empty", len(left))
	}
}

func TestGetTasksEmpty(t *testing.T) {
	if got := NewApp().GetTasks(); len(got) != 0 {
		t.Errorf("GetTasks = %d tasks; want empty", len(got))
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
		t.Fatalf("job %q did not finish", jobID)
	}

	if st, err := os.Stat(output); err != nil || st.Size() == 0 {
		t.Errorf("output file missing: err=%v", err)
	}
}

func TestGetUsageSmoke(t *testing.T) {
	snap := NewApp().GetUsage()
	if snap.MemTotalMB <= 0 {
		t.Errorf("MemTotalMB = %v, expected > 0", snap.MemTotalMB)
	}
	if snap.CPU < 0 || snap.CPU > 100 {
		t.Errorf("CPU = %v, fora de 0..100", snap.CPU)
	}
	if snap.GPU < -1 || snap.GPU > 100 {
		t.Errorf("GPU = %v, expected -1 ou 0..100", snap.GPU)
	}
}

func TestGetAdvice(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg": "#!/bin/sh\nexit 0\n",
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
	a := NewApp()

	if _, err := a.GetAdvice(compress.Job{}); err == nil {
		t.Error("expected erro sem inputPath")
	}
	if _, err := a.GetAdvice(compress.Job{InputPath: "in.mp4", PresetID: "nope"}); err == nil {
		t.Error("expected erro de preset desconhecido")
	}
	// a 2s 240p clip: plenty of budget → OK
	adv, err := a.GetAdvice(compress.Job{InputPath: "in.mp4", PresetID: "whatsapp-status", SizeMB: 10})
	if err != nil {
		t.Fatalf("GetAdvice: %v", err)
	}
	if !adv.OK {
		t.Errorf("clipe curto should estar ok: %+v", adv)
	}
}

func TestEstimateSize(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
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
	a := NewApp()

	if _, err := a.EstimateSize(compress.Job{}); err == nil {
		t.Error("expected erro sem inputPath")
	}
	if _, err := a.EstimateSize(compress.Job{InputPath: "in.mp4", PresetID: "nope"}); err == nil {
		t.Error("expected erro de preset desconhecido")
	}

	// modo size: o alvo carrega o headroom de 5% → ≈ 0.95 * 100 MB.
	res, err := a.EstimateSize(compress.Job{InputPath: "in.mp4", PresetID: "whatsapp-status", SizeMB: 100})
	if err != nil {
		t.Fatalf("EstimateSize (size): %v", err)
	}
	if !res.Available {
		t.Error("size mode should have an available estimate")
	}
	if math.Abs(res.TargetSizeMB-95) > 0.1 {
		t.Errorf("TargetSizeMB = %.3f; want ~95", res.TargetSizeMB)
	}

	// crf mode: no deterministic forecast yet → Available=false.
	crf, err := a.EstimateSize(compress.Job{InputPath: "in.mp4", PresetID: "youtube"})
	if err != nil {
		t.Fatalf("EstimateSize (crf): %v", err)
	}
	if crf.Available {
		t.Error("crf mode should not have a deterministic estimate")
	}
}

const encodersListApp = ` V....D libx264              libx264 H.264 / AVC / MPEG-4 AVC codec
 V..... h264_nvenc           NVIDIA NVENC H.264 encoder (codec h264)
 V..... hevc_nvenc           NVIDIA NVENC HEVC encoder (codec hevc)
 V..... h264_qsv             Intel Quick Sync H.264 encoder (codec h264)
 V..... hevc_qsv             Intel Quick Sync HEVC encoder (codec hevc)
`

// fakeDetectFFmpeg installs a scripted ffmpeg that lists encoders via
// FAKE_ENC_LIST and fails the probe of FAKE_PROBE_FAIL, returning the path.
func fakeDetectFFmpeg(t *testing.T, list, probeFail string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffmpeg")
	script := `#!/bin/sh
if [ "$2" = "-encoders" ]; then
	printf '%s' "$FAKE_ENC_LIST"
	exit 0
fi
if [ -n "$FAKE_PROBE_FAIL" ] && case "$*" in *"$FAKE_PROBE_FAIL"*) true ;; *) false ;; esac; then
	echo "probe falhou" >&2
	exit 1
fi
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_ENC_LIST", list)
	t.Setenv("FAKE_PROBE_FAIL", probeFail)
	cmdutil.SetOverride("ffmpeg", path)
	return path
}

func TestGetEncoders(t *testing.T) {
	skipOnWindows(t)
	fakeDetectFFmpeg(t, encodersListApp, "")
	a := newTestApp(t)

	avail, err := a.GetEncoders()
	if err != nil {
		t.Fatalf("GetEncoders: %v", err)
	}
	var found int
	for _, info := range avail.Hardware {
		if info.ID == "nvenc" || info.ID == "qsv" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("expected nvenc and qsv available, got %+v", avail.Hardware)
	}
	// segunda chamada vem do cache (sem erro)
	if _, err := a.GetEncoders(); err != nil {
		t.Errorf("cache should be available: %v", err)
	}
}

func TestRefreshEncodersReDetects(t *testing.T) {
	skipOnWindows(t)
	path := fakeDetectFFmpeg(t, encodersListApp, "")
	a := newTestApp(t)
	if _, err := a.GetEncoders(); err != nil {
		t.Fatalf("GetEncoders: %v", err)
	}

	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	after, err := a.RefreshEncoders()
	if err != nil {
		t.Fatalf("RefreshEncoders: %v", err)
	}
	if len(after.Hardware) != 0 {
		t.Errorf("after refresh expected no backends, got %+v", after.Hardware)
	}
}

func TestResolveEncoder(t *testing.T) {
	skipOnWindows(t)
	fakeDetectFFmpeg(t, encodersListApp, "")
	a := newTestApp(t)

	auto, err := a.resolveEncoder(compress.Job{Codec: "h265", Hardware: "auto"})
	if err != nil {
		t.Fatalf("resolveEncoder(auto): %v", err)
	}
	if auto.Hardware != "nvenc" {
		t.Errorf("auto should resolver para nvenc, got %q", auto.Hardware)
	}

	concrete, err := a.resolveEncoder(compress.Job{Hardware: "qsv"})
	if err != nil {
		t.Fatalf("resolveEncoder(qsv): %v", err)
	}
	if concrete.Hardware != "qsv" {
		t.Errorf("concrete hardware should not change, got %q", concrete.Hardware)
	}

	if _, err := a.resolveEncoder(compress.Job{Codec: "av1"}); err == nil || !strings.Contains(err.Error(), "unknown codec") {
		t.Errorf("invalid codec should error, got %v", err)
	}
	if _, err := a.resolveEncoder(compress.Job{Hardware: "cuda"}); err == nil || !strings.Contains(err.Error(), "unknown hardware encoder") {
		t.Errorf("invalid hardware should error, got %v", err)
	}

	cmdutil.SetOverride("ffmpeg", filepath.Join(t.TempDir(), "inexistente"))
	// The override is global and, if it leaks into the next test under shuffle,
	// it breaks
	// quem depende do ffmpeg real do PATH — limpa before de sair.
	t.Cleanup(cmdutil.ClearOverrides)
	b := newApp(nil)
	fallback, err := b.resolveEncoder(compress.Job{Hardware: "auto"})
	if err != nil {
		t.Fatalf("resolveEncoder(auto sem ffmpeg): %v", err)
	}
	if fallback.Hardware != "" {
		t.Errorf("auto sem encoders should virar software, got %q", fallback.Hardware)
	}
}

func TestCompressRejectsInvalidEncoder(t *testing.T) {
	a := newTestApp(t)
	if _, err := a.Compress(compress.Job{InputPath: "in.mp4", PresetID: "youtube", Codec: "av1"}); err == nil {
		t.Error("expected an invalid-codec error at enqueue")
	}
	if _, err := a.Compress(compress.Job{InputPath: "in.mp4", PresetID: "youtube", Hardware: "cuda"}); err == nil {
		t.Error("expected an invalid-hardware error at enqueue")
	}
}

func TestCompressRejectsInvalidAdjustments(t *testing.T) {
	a := newTestApp(t)
	if _, err := a.Compress(compress.Job{InputPath: "in.mp4", PresetID: "youtube", TrimStartSec: 30, TrimEndSec: 10}); err == nil {
		t.Error("expected erro de corte invertido no enqueue")
	}
	if _, err := a.Compress(compress.Job{InputPath: "in.mp4", PresetID: "youtube", Rotate: 45}); err == nil {
		t.Error("expected erro de invalid rotation no enqueue")
	}
}

func TestExtractAudioValidation(t *testing.T) {
	a := newTestApp(t)
	if _, err := a.ExtractAudio(audio.Spec{}); err == nil || !strings.Contains(err.Error(), "no video") {
		t.Errorf("empty spec should fail citing the video, got %v", err)
	}
	spec := audio.Spec{InputPath: "in.mp4", Codec: "ogg", OutputPath: "out.mp3"}
	if _, err := a.ExtractAudio(spec); err == nil || !strings.Contains(err.Error(), "format") {
		t.Errorf("invalid format should fail, got %v", err)
	}
	spec = audio.Spec{InputPath: "in.mp4", Codec: "mp3", OutputPath: ""}
	if _, err := a.ExtractAudio(spec); err == nil || !strings.Contains(err.Error(), "destination") {
		t.Errorf("sem destino should falhar, got %v", err)
	}
}

func TestExtractAudioEnqueuesAndRuns(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg": `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
mkdir -p "$(dirname "$out")"
printf 'audio' > "$out"
echo out_time_us=1000000
exit 0
`,
		"ffprobe": `#!/bin/sh
printf '%s\n' '{"streams":[
	{"codec_type":"video","codec_name":"h264","width":320,"height":240},
	{"codec_type":"audio","codec_name":"aac"}
],"format":{"duration":5.0}}'
`,
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	a := NewApp()
	events.SetEmitter(func(name string, data any) { a.jobs.Observe(name, data) })
	defer events.SetEmitter(nil)

	out := filepath.Join(t.TempDir(), "in-audio.mp3")
	jobID, err := a.ExtractAudio(audio.Spec{InputPath: "in.mp4", Codec: "mp3", BitrateKbps: 128, OutputPath: out})
	if err != nil || jobID == "" {
		t.Fatalf("ExtractAudio: jobID=%q err=%v", jobID, err)
	}

	s := waitTaskTerminal(t, a, jobID)
	if s.State != compress.StateDone {
		t.Errorf("estado final = %q; want done (task=%+v)", s.State, s)
	}
	if s.Kind != "audio" {
		t.Errorf("kind = %q; want audio", s.Kind)
	}
	if s.OutputPath != out {
		t.Errorf("outputPath = %q; want %q", s.OutputPath, out)
	}
	if s.InputPath != "in.mp4" {
		t.Errorf("inputPath = %q; want the extraction path", s.InputPath)
	}
	if st, err := os.Stat(out); err != nil || st.Size() == 0 {
		t.Errorf("audio file missing: err=%v", err)
	}
}

func TestExtractAudioVideoWithoutAudio(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffprobe": `#!/bin/sh
printf '%s\n' '{"streams":[
	{"codec_type":"video","codec_name":"h264","width":320,"height":240}
],"format":{"duration":5.0}}'
`,
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	a := NewApp()
	out := filepath.Join(t.TempDir(), "in-audio.mp3")
	jobID, err := a.ExtractAudio(audio.Spec{InputPath: "in.mp4", Codec: "mp3", OutputPath: out})
	if err != nil {
		t.Fatalf("ExtractAudio: %v", err)
	}
	s := waitTaskTerminal(t, a, jobID)
	if s.State != compress.StateError || !strings.Contains(s.Error, "has no audio track") {
		t.Errorf("estado final = %+v; want erro de faixa ausente", s)
	}
}

func TestSuggestAudioOutput(t *testing.T) {
	a := newTestApp(t)

	if _, err := a.SuggestAudioOutput("", "mp3"); err == nil {
		t.Error("sem entrada should falhar")
	}
	if _, err := a.SuggestAudioOutput("/tmp/video.mp4", "ogg"); err == nil {
		t.Error("invalid format should fail")
	}

	got, err := a.SuggestAudioOutput(filepath.Join("videos", "meu video.mov"), "flac")
	if err != nil {
		t.Fatalf("SuggestAudioOutput: %v", err)
	}
	// Without outputDir in the config, the destination is the video folder.
	want := filepath.Join("videos", "meu video-audio.flac")
	if got != want {
		t.Errorf("SuggestAudioOutput = %q, want %q", got, want)
	}

	saved := config.Defaults()
	saved.OutputDir = filepath.Join("destino")
	if err := a.SaveConfig(saved); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	got, err = a.SuggestAudioOutput(filepath.Join("videos", "a.mp4"), "mp3")
	if err != nil {
		t.Fatalf("SuggestAudioOutput com outputDir: %v", err)
	}
	if want := filepath.Join("destino", "a-audio.mp3"); got != want {
		t.Errorf("SuggestAudioOutput = %q, want %q", got, want)
	}

	custom := config.Defaults()
	custom.OutputDir = ""
	custom.AudioSuffix = "-triha"
	if err := a.SaveConfig(custom); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	got, err = a.SuggestAudioOutput("/tmp/x.mkv", "opus")
	if err != nil {
		t.Fatalf("SuggestAudioOutput com sufixo custom: %v", err)
	}
	if want := filepath.Join("/tmp", "x-triha.opus"); got != want {
		t.Errorf("SuggestAudioOutput = %q, want %q", got, want)
	}
}

func TestTranscribeVideoValidation(t *testing.T) {
	a := newTestApp(t)
	if _, err := a.TranscribeVideo(transcribe.Spec{}); err == nil || !strings.Contains(err.Error(), "no video") {
		t.Errorf("empty spec should fail citing the video, got %v", err)
	}
	if _, err := a.TranscribeVideo(transcribe.Spec{InputPath: "in.mp4"}); err == nil {
		t.Error("missing output base should fail")
	}
}

func TestTranscribeVideoEnqueuesAndRuns(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg": `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
mkdir -p "$(dirname "$out")"
printf 'wav' > "$out"
exit 0
`,
		"ffprobe": `#!/bin/sh
printf '%s\n' '{"streams":[
	{"codec_type":"video","codec_name":"h264","width":320,"height":240},
	{"codec_type":"audio","codec_name":"aac"}
],"format":{"duration":5.0}}'
`,
		"whisper-cli": `#!/bin/sh
base=""
prev=""
for a in "$@"; do
  if [ "$prev" = "-of" ]; then base="$a"; fi
  prev="$a"
done
printf '%s\n' '{"transcription":[{"offsets":{"from":0,"to":2000},"text":"hello world"}]}' > "$base.json"
exit 0
`,
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	transcribe.ResetDetection()
	t.Cleanup(transcribe.ResetDetection)

	a := NewApp()
	events.SetEmitter(func(name string, data any) { a.jobs.Observe(name, data) })
	defer events.SetEmitter(nil)

	base := filepath.Join(t.TempDir(), "in-transcript")
	jobID, err := a.TranscribeVideo(transcribe.Spec{
		InputPath: "in.mp4", OutputBase: base, Engine: "auto", Language: "auto", Format: "all",
		ModelPath: "/models/ggml.bin",
	})
	if err != nil || jobID == "" {
		t.Fatalf("TranscribeVideo: jobID=%q err=%v", jobID, err)
	}

	s := waitTaskTerminal(t, a, jobID)
	if s.State != compress.StateDone {
		t.Fatalf("estado final = %q; want done (task=%+v)", s.State, s)
	}
	if s.Kind != events.KindTranscribe {
		t.Errorf("kind = %q; want transcribe", s.Kind)
	}
	for _, ext := range []string{"srt", "vtt", "txt"} {
		if _, err := os.Stat(base + "." + ext); err != nil {
			t.Errorf("missing %s file: %v", ext, err)
		}
	}
}

func TestSuggestTranscribeOutput(t *testing.T) {
	a := newTestApp(t)
	if _, err := a.SuggestTranscribeOutput(""); err == nil {
		t.Error("sem entrada should falhar")
	}

	got, err := a.SuggestTranscribeOutput(filepath.Join("videos", "meu video.mov"))
	if err != nil {
		t.Fatalf("SuggestTranscribeOutput: %v", err)
	}
	if want := filepath.Join("videos", "meu video-transcript"); got != want {
		t.Errorf("SuggestTranscribeOutput = %q, want %q", got, want)
	}

	saved := config.Defaults()
	saved.OutputDir = "destino"
	saved.TranscribeSuffix = "-legenda"
	if err := a.SaveConfig(saved); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	got, err = a.SuggestTranscribeOutput("/tmp/x.mkv")
	if err != nil {
		t.Fatalf("SuggestTranscribeOutput com overrides: %v", err)
	}
	if want := filepath.Join("destino", "x-legenda"); got != want {
		t.Errorf("SuggestTranscribeOutput = %q, want %q", got, want)
	}
}

func TestGetAndRefreshTranscribers(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "whisper-cli"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Cleanup(transcribe.ResetDetection)

	a := newTestApp(t)
	got := a.RefreshTranscribers()
	if len(got) != 1 || got[0] != transcribe.EngineWhisper {
		t.Errorf("RefreshTranscribers = %v; want [whispercpp]", got)
	}
}

type notifyRecorder struct {
	ch chan string
}

func (n *notifyRecorder) Notify(title, message string) error {
	n.ch <- title + ": " + message
	return nil
}

func fakeFFprobeScript() string {
	return `#!/bin/sh
printf '%s\n' '{"streams":[{"codec_type":"video","codec_name":"h264","width":320,"height":240,"duration":2.0}],"format":{"duration":2.0}}'
`
}

func newNotificationApp(t *testing.T) *App {
	t.Helper()
	a := newTestApp(t)
	// Observe-only emitter: the real startup() routes through the Wails runtime
	// context, which aborts the test binary on invalid lifecycle contexts.
	events.SetEmitter(func(name string, data any) {
		a.jobs.Observe(name, data)
	})
	t.Cleanup(func() { events.SetEmitter(nil) })
	return a
}

func TestNotifyOnDone(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg":  "#!/bin/sh\nexit 0\n",
		"ffprobe": fakeFFprobeScript(),
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	rec := &notifyRecorder{ch: make(chan string, 4)}
	notify.SetSender(rec)
	t.Cleanup(func() { notify.SetSender(nil) })

	a := newNotificationApp(t)
	cfg := config.Defaults()
	cfg.NotifyOnDone = true
	if err := a.store.Save(cfg); err != nil {
		t.Fatal(err)
	}

	jobID, err := a.Compress(compress.Job{
		InputPath:  "in.mp4",
		OutputPath: filepath.Join(t.TempDir(), "out.mp4"),
		PresetID:   "whatsapp-status",
	})
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if s := waitTaskTerminal(t, a, jobID); s.State != compress.StateDone {
		t.Fatalf("final state = %q; want done", s.State)
	}
	select {
	case got := <-rec.ch:
		if !strings.Contains(got, "vidctl") || !strings.Contains(got, "in.mp4") || !strings.Contains(got, "finished") {
			t.Errorf("notification = %q; want vidctl/in.mp4 finished", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no notification emitted on done")
	}
}

func TestNotifyOnError(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg":  "#!/bin/sh\nexit 1\n",
		"ffprobe": fakeFFprobeScript(),
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	rec := &notifyRecorder{ch: make(chan string, 4)}
	notify.SetSender(rec)
	t.Cleanup(func() { notify.SetSender(nil) })

	a := newNotificationApp(t)
	cfg := config.Defaults()
	cfg.NotifyOnDone = true
	if err := a.store.Save(cfg); err != nil {
		t.Fatal(err)
	}

	jobID, err := a.Compress(compress.Job{
		InputPath: "in.mp4",
		PresetID:  "whatsapp-status",
	})
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if s := waitTaskTerminal(t, a, jobID); s.State != compress.StateError {
		t.Fatalf("final state = %q; want error", s.State)
	}
	select {
	case got := <-rec.ch:
		if !strings.Contains(got, "in.mp4") || !strings.Contains(got, "failed") {
			t.Errorf("notification = %q; want in.mp4 failed", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no notification emitted on error")
	}
}

func TestOpenFolderOnDone(t *testing.T) {
	skipOnWindows(t)
	if runtime.GOOS != "linux" {
		t.Skip("the fake xdg-open only covers the Linux opener")
	}
	marker := filepath.Join(t.TempDir(), "opened-dir")
	dir := t.TempDir()
	scripts := map[string]string{
		"ffmpeg":   "#!/bin/sh\nexit 0\n",
		"ffprobe":  fakeFFprobeScript(),
		"xdg-open": "#!/bin/sh\nprintf '%s' \"$1\" > '" + marker + "'\n",
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	a := newNotificationApp(t)
	cfg := config.Defaults()
	cfg.OpenFolderOnDone = true
	if err := a.store.Save(cfg); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	jobID, err := a.Compress(compress.Job{
		InputPath:  "in.mp4",
		OutputPath: filepath.Join(outDir, "out.mp4"),
		PresetID:   "whatsapp-status",
	})
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if s := waitTaskTerminal(t, a, jobID); s.State != compress.StateDone {
		t.Fatalf("final state = %q; want done", s.State)
	}

	deadline := time.Now().Add(5 * time.Second)
	var got []byte
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(marker)
		if err == nil {
			got = data
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if string(got) != outDir {
		t.Errorf("xdg-open received %q; want %q", got, outDir)
	}
}

func TestSystemThemeBinding(t *testing.T) {
	a := NewApp()
	if got := a.SystemTheme(); got != "dark" && got != "light" {
		t.Errorf("SystemTheme() = %q, want dark|light", got)
	}
}

type fakeUpdateChecker struct {
	info  update.Info
	times int
}

func (f *fakeUpdateChecker) Check(current string) (update.Info, error) {
	f.times++
	return f.info, nil
}

func TestCheckUpdateBinding(t *testing.T) {
	a := newTestApp(t)
	a.updates = &fakeUpdateChecker{info: update.Info{Ok: true, HasUpdate: true, Latest: "v9.9.9"}}
	got, err := a.CheckUpdate()
	if err != nil {
		t.Fatalf("CheckUpdate: %v", err)
	}
	if !got.Ok || !got.HasUpdate || got.Latest != "v9.9.9" {
		t.Errorf("CheckUpdate = %+v; want ok+hasUpdate v9.9.9", got)
	}
	if fc, ok := a.updates.(*fakeUpdateChecker); ok && fc.times != 1 {
		t.Errorf("checker called %d times; want 1", fc.times)
	}
}

func TestCheckUpdateSilentOnFailure(t *testing.T) {
	a := newTestApp(t)
	a.updates = &fakeUpdateChecker{info: update.Info{Ok: false}}
	got, err := a.CheckUpdate()
	if err != nil {
		t.Fatalf("CheckUpdate: %v", err)
	}
	if got.Ok {
		t.Errorf("CheckUpdate = %+v; want silent no-update", got)
	}
}

func withPresetStore(t *testing.T, a *App) *presets.Store {
	t.Helper()
	s, err := presets.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("presets.NewStore: %v", err)
	}
	a.presetStore = s
	presets.SetCustomSource(s.LoadAll)
	t.Cleanup(func() { presets.SetCustomSource(nil) })
	return s
}

func TestAddCustomPresetBinding(t *testing.T) {
	a := newTestApp(t)
	withPresetStore(t, a)

	all, err := a.AddCustomPreset(presets.Preset{Name: "Lecture HD", Mode: "size", SizeMB: 25, AudioBitrate: "128k", Codec: "h265"})
	if err != nil {
		t.Fatalf("AddCustomPreset: %v", err)
	}
	var added *presets.Preset
	for i := range all {
		if all[i].Name == "Lecture HD" {
			added = &all[i]
		}
	}
	if added == nil || !strings.HasPrefix(added.ID, presets.CustomIDPrefix) {
		t.Fatalf("added preset not in merged list: %+v", all)
	}
	if _, ok := presets.ByID(added.ID); !ok {
		t.Error("custom preset should resolve via ByID")
	}
	if p, ok := compress.EffectivePreset(compress.Job{PresetID: added.ID}); !ok || p.SizeMB != 25 || p.Codec != "h265" {
		t.Errorf("EffectivePreset(custom) = %+v, ok=%v; want the custom preset", p, ok)
	}
}

func TestUpdateCustomPresetBinding(t *testing.T) {
	a := newTestApp(t)
	withPresetStore(t, a)
	all, err := a.AddCustomPreset(presets.Preset{Name: "A", Mode: "crf", CRF: 20, AudioBitrate: "96k"})
	if err != nil {
		t.Fatal(err)
	}
	id := all[len(all)-1].ID

	updated, err := a.UpdateCustomPreset(id, presets.Preset{Name: "B", Mode: "crf", CRF: 28, AudioBitrate: "128k"})
	if err != nil {
		t.Fatalf("UpdateCustomPreset: %v", err)
	}
	found := false
	for _, p := range updated {
		if p.ID == id && p.Name == "B" && p.CRF == 28 {
			found = true
		}
	}
	if !found {
		t.Errorf("updated preset not found: %+v", updated)
	}
}

func TestDeleteCustomPresetBinding(t *testing.T) {
	a := newTestApp(t)
	withPresetStore(t, a)
	all, err := a.AddCustomPreset(presets.Preset{Name: "Temp", Mode: "size", SizeMB: 5})
	if err != nil {
		t.Fatal(err)
	}
	id := all[len(all)-1].ID
	left, err := a.DeleteCustomPreset(id)
	if err != nil {
		t.Fatalf("DeleteCustomPreset: %v", err)
	}
	for _, p := range left {
		if p.ID == id {
			t.Errorf("deleted preset still present: %+v", left)
		}
	}
}

func TestCustomPresetBindingsWithoutStore(t *testing.T) {
	a := newTestApp(t)
	a.presetStore = nil
	if _, err := a.AddCustomPreset(presets.Preset{Name: "X", Mode: "size", SizeMB: 1}); err == nil {
		t.Error("Add without a store should fail")
	}
	if got := a.GetCustomPresets(); got != nil {
		t.Errorf("GetCustomPresets without store = %+v; want nil", got)
	}
}
