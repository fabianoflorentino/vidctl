package compress

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/estimate"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/split"
)

func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell bins não executam no Windows")
	}
}

const fakeFFmpegOK = `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
case "$out" in /dev/null) ;; *) printf 'hi' > "$out" ;; esac
echo out_time_us=1000000
exit 0
`

const fakeFFmpegFail = "#!/bin/sh\nexit 1\n"

const fakeFFprobe = `#!/bin/sh
printf '%s\n' '{"streams":[
	{"codec_type":"video","codec_name":"h264","width":320,"height":240,"duration":2.0},
	{"codec_type":"audio","codec_name":"aac"}
],"format":{"duration":2.0}}'
`

func fakeToolchain(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	write := func(name, script string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("ffmpeg", fakeFFmpegOK)
	write("ffprobe", fakeFFprobe)
	t.Setenv("PATH", dir)
}

type rec struct {
	name string
	data any
}

func captureCompressEvents(t *testing.T) *[]rec {
	t.Helper()
	var out []rec
	events.SetEmitter(func(name string, data any) { out = append(out, rec{name, data}) })
	t.Cleanup(func() { events.SetEmitter(nil) })
	return &out
}

func findErr(got *[]rec) string {
	for _, r := range *got {
		if r.name == "compress:error" {
			return r.data.(events.ErrorEvent).Error
		}
	}
	return ""
}

func TestFFmpegPathMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := ffmpegPath(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFFmpegPathUsesOverride(t *testing.T) {
	t.Cleanup(cmdutil.ClearOverrides)
	t.Setenv("PATH", t.TempDir())

	custom := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmdutil.SetOverride("ffmpeg", custom)

	got, err := ffmpegPath()
	if err != nil {
		t.Fatalf("ffmpegPath com override: %v", err)
	}
	if got != custom {
		t.Errorf("ffmpegPath() = %q, want %q", got, custom)
	}
}

func TestFFmpegPathReportsBrokenOverride(t *testing.T) {
	t.Cleanup(cmdutil.ClearOverrides)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte(fakeFFmpegOK), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	cmdutil.SetOverride("ffmpeg", filepath.Join(dir, "nao-existe"))

	_, err := ffmpegPath()
	if err == nil {
		t.Fatal("ffmpegPath deveria falhar com override inválido")
	}
	if !errors.Is(err, cmdutil.ErrNotConfigured) {
		t.Errorf("erro %v não é ErrNotConfigured", err)
	}
	if !strings.Contains(err.Error(), "nao-existe") {
		t.Errorf("mensagem %q deveria citar o caminho configurado", err)
	}
}

func TestRunUsesOverrideInsteadOfPath(t *testing.T) {
	skipOnWindows(t)
	t.Cleanup(cmdutil.ClearOverrides)

	// O ffmpeg do PATH falha; o configurado funciona. Se o override for
	// respeitado, o job conclui.
	pathDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pathDir, "ffmpeg"), []byte(fakeFFmpegFail), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pathDir, "ffprobe"), []byte(fakeFFprobe), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", pathDir)

	customDir := t.TempDir()
	custom := filepath.Join(customDir, "ffmpeg")
	if err := os.WriteFile(custom, []byte(fakeFFmpegOK), 0o755); err != nil {
		t.Fatal(err)
	}
	cmdutil.SetOverride("ffmpeg", custom)

	got := captureCompressEvents(t)
	out := filepath.Join(t.TempDir(), "out.mp4")
	Run(t.Context(), "job-override-ok", Job{
		InputPath:  filepath.Join(customDir, "in.mp4"),
		OutputPath: out,
		PresetID:   "whatsapp-status",
		SizeMB:     5,
	})

	if msg := findErr(got); msg != "" {
		t.Fatalf("job falhou apesar do override válido: %s", msg)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("saída não criada: %v", err)
	}
}

func TestRunFailsWithBrokenOverride(t *testing.T) {
	skipOnWindows(t)
	t.Cleanup(cmdutil.ClearOverrides)

	// O ffmpeg do PATH funciona, mas o override aponta para arquivo nenhum:
	// o app não pode cair silenciosamente no binário do PATH.
	pathDir := t.TempDir()
	for name, script := range map[string]string{"ffmpeg": fakeFFmpegOK, "ffprobe": fakeFFprobe} {
		if err := os.WriteFile(filepath.Join(pathDir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", pathDir)
	cmdutil.SetOverride("ffmpeg", filepath.Join(t.TempDir(), "nao-existe"))

	got := captureCompressEvents(t)
	Run(t.Context(), "job-override-bad", Job{
		InputPath:  filepath.Join(t.TempDir(), "in.mp4"),
		OutputPath: filepath.Join(t.TempDir(), "out.mp4"),
		PresetID:   "whatsapp-status",
		SizeMB:     5,
	})

	if msg := findErr(got); msg == "" {
		t.Fatal("job deveria falhar com override inválido")
	} else if !strings.Contains(msg, "nao-existe") {
		t.Errorf("mensagem %q deveria citar o caminho configurado", msg)
	}
}

func TestPassLogManagement(t *testing.T) {
	base := passLogFile("job-x")
	if !strings.Contains(base, "vidctl-job-x") {
		t.Errorf("passLogFile inesperado: %s", base)
	}
	for _, f := range []string{base, base + "-0.log", base + "-0.log.mbtree"} {
		if err := os.WriteFile(f, []byte("z"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	removePassLogs("job-x")
	for _, f := range []string{base, base + "-0.log", base + "-0.log.mbtree"} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("esperava remoção de %s", f)
		}
	}
}

func TestBuildSizePasses(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	info := &media.Info{DurationSec: 80, HasAudio: true}
	p := presets.Preset{Mode: "size", SizeMB: 10, AudioBitrate: "96k", CRF: 23}

	p1, p2, err := buildSizePasses(context.Background(), "job-s", Job{InputPath: "in.mp4", OutputPath: "out.mp4"}, p, info, split.Segment{Index: 1, EndSec: 80})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsStr(p1.Args, "-pass", "1") {
		t.Errorf("pass1 deve ter -pass 1, args: %v", p1.Args)
	}
	if !containsStr(p2.Args, "-pass", "2") {
		t.Errorf("pass2 deve ter -pass 2, args: %v", p2.Args)
	}

	budget, err := estimate.BudgetFor(p, info, 80)
	if err != nil {
		t.Fatalf("erro de orçamento inesperado: %v", err)
	}
	if !containsStr(p2.Args, "-b:v", strconv.Itoa(budget.VideoBitrate)) {
		t.Errorf("pass2 deve usar o bitrate do estimate (%d), args: %v", budget.VideoBitrate, p2.Args)
	}

	t.Run("missing ffmpeg", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		if _, _, err := buildSizePasses(context.Background(), "j", Job{}, p, info, split.Segment{}); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("budget error", func(t *testing.T) {
		fakeToolchain(t)
		info := &media.Info{DurationSec: 3600, HasAudio: true}
		p := presets.Preset{Mode: "size", SizeMB: 1, AudioBitrate: "96k"}
		if _, _, err := buildSizePasses(context.Background(), "j", Job{}, p, info, split.Segment{Index: 1, EndSec: 3600}); err == nil {
			t.Fatal("expected budget error, got nil")
		}
	})
}

func TestBuildCrfPass(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	p := presets.Preset{CRF: 23, AudioBitrate: "96k"}

	withAudio := buildCrfPass(context.Background(), Job{InputPath: "in.mp4", OutputPath: "out.mp4"}, p, &media.Info{HasAudio: true}, split.Segment{})
	if !containsStr(withAudio.Args, "-c:a", "aac") {
		t.Errorf("com áudio deve incluir -c:a aac, args: %v", withAudio.Args)
	}

	noAudio := buildCrfPass(context.Background(), Job{InputPath: "in.mp4", OutputPath: "out.mp4"}, p, &media.Info{HasAudio: false}, split.Segment{})
	if containsStr(noAudio.Args, "-c:a", "aac") {
		t.Errorf("sem áudio não deve incluir -c:a aac, args: %v", noAudio.Args)
	}

	t.Run("missing ffmpeg ignores error", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		cmd := buildCrfPass(context.Background(), Job{}, p, &media.Info{HasAudio: false}, split.Segment{})
		if cmd.Args[0] != "" {
			t.Errorf("binário deveria estar vazio, got %q", cmd.Args[0])
		}
	})
}

func TestExecuteProgress(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"),
		[]byte("#!/bin/sh\necho out_time_us=500000\necho out_time_us=2000000\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	got := captureCompressEvents(t)
	cmd := exec.CommandContext(context.Background(), "ffmpeg")
	if err := execute(context.Background(), cmd, "job-e", "pass1/2", 2.0, 0, 100); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	var percents []float64
	for _, r := range *got {
		if p, ok := r.data.(events.ProgressEvent); ok {
			percents = append(percents, p.Percent)
		}
	}
	if len(percents) != 2 {
		t.Fatalf("esperava 2 eventos de progresso, got %d", len(percents))
	}
	if percents[0] != 25 || percents[1] != 99.9 {
		t.Errorf("progressos inesperados: %v", percents)
	}
}

func TestExecuteFails(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	cmd := exec.CommandContext(context.Background(), "ffmpeg")
	if err := execute(context.Background(), cmd, "job-f", "pass1/2", 2.0, 0, 100); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunEmptyPaths(t *testing.T) {
	got := captureCompressEvents(t)
	Run(context.Background(), "j1", Job{})
	if err := findErr(got); !strings.Contains(err, "pode ser vazio") {
		t.Errorf("erro inesperado: %q", err)
	}
}

func TestRunSameInputOutput(t *testing.T) {
	got := captureCompressEvents(t)
	Run(context.Background(), "j2", Job{InputPath: "x.mp4", OutputPath: "x.mp4"})
	if err := findErr(got); !strings.Contains(err, "igual ao de entrada") {
		t.Errorf("erro inesperado: %q", err)
	}
}

func TestRunFFprobeMissing(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte(fakeFFmpegOK), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	got := captureCompressEvents(t)
	Run(context.Background(), "j3", Job{InputPath: "in.mp4", OutputPath: "out.mp4", PresetID: "whatsapp-status"})
	if err := findErr(got); !strings.Contains(err, "ffprobe não encontrado") {
		t.Errorf("erro inesperado: %q", err)
	}
}

func TestRunUnknownPreset(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	got := captureCompressEvents(t)
	Run(context.Background(), "j4", Job{InputPath: "in.mp4", OutputPath: "out.mp4", PresetID: "nope"})
	if err := findErr(got); !strings.Contains(err, "preset desconhecido") {
		t.Errorf("erro inesperado: %q", err)
	}
}

func TestRunMkdirFailure(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	blocker := filepath.Join(t.TempDir(), "bloqueio")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := captureCompressEvents(t)
	Run(context.Background(), "j5", Job{InputPath: "in.mp4", OutputPath: filepath.Join(blocker, "out.mp4"), PresetID: "whatsapp-status"})
	if err := findErr(got); !strings.Contains(err, "falha ao criar pasta") {
		t.Errorf("erro inesperado: %q", err)
	}
}

func TestRunSizeSuccess(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	output := filepath.Join(t.TempDir(), "out.mp4")
	got := captureCompressEvents(t)
	Run(context.Background(), "j6", Job{InputPath: "in.mp4", OutputPath: output, PresetID: "whatsapp-status"})

	var stages []string
	for _, r := range *got {
		if p, ok := r.data.(events.ProgressEvent); ok {
			stages = append(stages, p.Stage)
		}
	}
	if idx1, idx2 := indexStage(stages, "pass1/2"), indexStage(stages, "pass2/2"); idx1 < 0 || idx2 < 0 || idx1 > idx2 {
		t.Errorf("stages inesperados (pass1 antes de pass2): %v", stages)
	}
	var done int
	for _, r := range *got {
		if r.name == "compress:done" {
			done++
			if d := r.data.(events.DoneEvent); d.OutputPath != output || d.SizeBytes <= 0 {
				t.Errorf("done inesperado: %+v", d)
			}
		}
	}
	if done != 1 {
		t.Errorf("esperava 1 done, got %d", done)
	}
	if e := findErr(got); e != "" {
		t.Errorf("erro inesperado: %q", e)
	}
}

func TestRunSizePass1Failure(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	for name, script := range map[string]string{"ffmpeg": fakeFFmpegFail, "ffprobe": fakeFFprobe} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	got := captureCompressEvents(t)
	Run(context.Background(), "j7", Job{InputPath: "in.mp4", OutputPath: filepath.Join(t.TempDir(), "out.mp4"), PresetID: "whatsapp-status"})
	if err := findErr(got); !strings.Contains(err, "pass 1 falhou") {
		t.Errorf("erro inesperado: %q", err)
	}
}

func TestRunCrfSuccess(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	output := filepath.Join(t.TempDir(), "out.mp4")
	got := captureCompressEvents(t)
	Run(context.Background(), "j8", Job{InputPath: "in.mp4", OutputPath: output, PresetID: "youtube", CRF: 30})

	var done int
	for _, r := range *got {
		if r.name == "compress:done" {
			done++
			if d := r.data.(events.DoneEvent); d.OutputPath != output || d.SizeBytes <= 0 {
				t.Errorf("done inesperado: %+v", d)
			}
		}
	}
	if done != 1 {
		t.Errorf("esperava 1 done, got %d", done)
	}
	if e := findErr(got); e != "" {
		t.Errorf("erro inesperado: %q", e)
	}
}

func TestRunCrfFailure(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	for name, script := range map[string]string{"ffmpeg": fakeFFmpegFail, "ffprobe": fakeFFprobe} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	got := captureCompressEvents(t)
	Run(context.Background(), "j9", Job{InputPath: "in.mp4", OutputPath: filepath.Join(t.TempDir(), "out.mp4"), PresetID: "youtube"})
	if err := findErr(got); !strings.Contains(err, "falha ao comprimir") {
		t.Errorf("erro inesperado: %q", err)
	}
}

func TestRunCanceled(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := captureCompressEvents(t)
	Run(ctx, "j10", Job{InputPath: "in.mp4", OutputPath: filepath.Join(t.TempDir(), "out.mp4"), PresetID: "whatsapp-status"})
	if len(*got) != 1 {
		t.Fatalf("esperava apenas o progresso inicial, got %+v", *got)
	}
	progress, ok := (*got)[0].data.(events.ProgressEvent)
	if !ok || progress.Stage != "pass1/2" || progress.Percent != 0 {
		t.Errorf("progresso inicial esperado pass1/2=0, got %+v", *got)
	}
	for _, r := range *got {
		if r.name == "compress:error" || r.name == "compress:done" {
			t.Errorf("cancelamento não deve emitir error/done, got %+v", *got)
		}
	}
}

const fakeFFprobeLong = `#!/bin/sh
printf '%s\n' '{"streams":[
	{"codec_type":"video","codec_name":"h264","width":320,"height":240,"duration":180.0},
	{"codec_type":"audio","codec_name":"aac"}
],"format":{"duration":180.0}}'
`

const fakeFFmpegRecord = `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
case "$out" in /dev/null) ;; *) printf 'hi' > "$out" ;; esac
printf '%s\n' "$*" >> "$ARGS_LOG"
echo out_time_us=1000000
exit 0
`

func TestRunSizeSplit(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, script := range map[string]string{"ffmpeg": fakeFFmpegRecord, "ffprobe": fakeFFprobeLong} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	argsLog := filepath.Join(dir, "args.log")
	t.Setenv("ARGS_LOG", argsLog)

	output := filepath.Join(dir, "out.mp4")
	got := captureCompressEvents(t)
	Run(context.Background(), "j-split", Job{
		InputPath:  "in.mp4",
		OutputPath: output,
		PresetID:   "whatsapp-status",
		Split:      &split.Spec{Parts: 3},
	})

	var donePaths []string
	var stages []string
	for _, r := range *got {
		switch d := r.data.(type) {
		case events.DoneEvent:
			donePaths = append(donePaths, filepath.Base(d.OutputPath))
			if d.SizeBytes <= 0 {
				t.Errorf("done %s com SizeBytes <= 0", d.OutputPath)
			}
		case events.ProgressEvent:
			stages = append(stages, d.Stage)
		}
	}
	if e := findErr(got); e != "" {
		t.Fatalf("erro inesperado: %q", e)
	}
	want := []string{"out-part1.mp4", "out-part2.mp4", "out-part3.mp4"}
	if len(donePaths) != 3 || donePaths[0] != want[0] || donePaths[1] != want[1] || donePaths[2] != want[2] {
		t.Fatalf("dones = %v, want %v", donePaths, want)
	}
	if len(stages) == 0 || !strings.Contains(stages[len(stages)-1], "parte 3/3") {
		t.Errorf("última stage deveria indicar parte 3/3, got %v", stages)
	}

	raw, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 6 {
		t.Fatalf("esperava 6 comandos (2 por parte), got %d: %q", len(lines), string(raw))
	}
	if !strings.Contains(lines[0], "-ss 0.000 -to 60.000 -i in.mp4") {
		t.Errorf("seek antes de -i ausente no pass1 da parte 1: %q", lines[0])
	}
	if !strings.Contains(lines[2], "-ss 60.000 -to 120.000") {
		t.Errorf("seek da parte 2 ausente: %q", lines[2])
	}
	if !strings.Contains(lines[0], "vidctl-j-split-p1 ") || !strings.Contains(lines[2], "vidctl-j-split-p2 ") {
		t.Errorf("passlogfile deveria diferir por parte: %q / %q", lines[0], lines[2])
	}
	part1 := strings.TrimSuffix(output, ".mp4") + "-part1.mp4"
	part2 := strings.TrimSuffix(output, ".mp4") + "-part2.mp4"
	if !strings.HasSuffix(lines[1], part1) || !strings.HasSuffix(lines[3], part2) {
		t.Errorf("saídas por parte ausentes: %q / %q", lines[1], lines[3])
	}
}

func TestRunSplitInvalid(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, script := range map[string]string{"ffmpeg": fakeFFmpegOK, "ffprobe": fakeFFprobeLong} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	got := captureCompressEvents(t)
	Run(context.Background(), "j-bad-split", Job{
		InputPath:  "in.mp4",
		OutputPath: filepath.Join(t.TempDir(), "out.mp4"),
		PresetID:   "whatsapp-status",
		Split:      &split.Spec{Parts: 300},
	})
	if e := findErr(got); !strings.Contains(e, "máximo") {
		t.Errorf("esperava erro de máximo de partes, got %q", e)
	}
}

func TestRunNoSplitKeepsSingleDone(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	dir := t.TempDir()
	argsLog := filepath.Join(dir, "args.log")
	t.Setenv("ARGS_LOG", argsLog)
	bin := t.TempDir()
	for name, script := range map[string]string{"ffmpeg": fakeFFmpegRecord, "ffprobe": fakeFFprobe} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)

	got := captureCompressEvents(t)
	Run(context.Background(), "j-nosplit", Job{
		InputPath:  "in.mp4",
		OutputPath: filepath.Join(dir, "out.mp4"),
		PresetID:   "whatsapp-status",
	})
	var done int
	for _, r := range *got {
		if r.name == "compress:done" {
			done++
		}
	}
	if done != 1 {
		t.Fatalf("esperava 1 done, got %d", done)
	}
	raw, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "-ss") {
		t.Errorf("sem split não deve conter -ss: %q", string(raw))
	}
}

func containsStr(args []string, wanted ...string) bool {
	for i := 0; i+len(wanted) <= len(args); i++ {
		match := true
		for j := range wanted {
			if args[i+j] != wanted[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func indexStage(stages []string, s string) int {
	for i := range stages {
		if stages[i] == s {
			return i
		}
	}
	return -1
}
