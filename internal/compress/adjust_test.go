package compress

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/encode"
	"github.com/fabianoflorentino/vidctl/internal/estimate"
	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/split"
)

func TestBuildVideoFilter(t *testing.T) {
	tests := []struct {
		name string
		job  Job
		want string
	}{
		{"padrão", Job{}, defaultScaleFilter},
		{"original omite scale", Job{Scale: ScaleOriginal}, ""},
		{"custom", Job{Scale: "1920x1080"}, "scale=1920x1080:force_original_aspect_ratio=decrease"},
		{"rotação 90", Job{Rotate: 90}, "transpose=1," + defaultScaleFilter},
		{"rotação 180", Job{Rotate: 180}, "transpose=2," + defaultScaleFilter},
		{"rotação 270", Job{Rotate: 270}, "transpose=3," + defaultScaleFilter},
		{"fps", Job{FPS: 30}, defaultScaleFilter + ",fps=30.000"},
		{"ordem transpose→scale→fps", Job{Rotate: 90, Scale: "640x480", FPS: 24},
			"transpose=1,scale=640x480:force_original_aspect_ratio=decrease,fps=24.000"},
		{"original com fps", Job{Scale: ScaleOriginal, FPS: 15}, "fps=15.000"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildVideoFilter(tc.job); got != tc.want {
				t.Errorf("buildVideoFilter() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	valid := []Job{
		{},
		{TrimStartSec: 10, TrimEndSec: 20},
		{TrimStartSec: 10},
		{TrimEndSec: 20},
		{FPS: 30},
		{Rotate: 90}, {Rotate: 180}, {Rotate: 270},
		{Scale: ScaleOriginal},
		{Scale: "1280x720"},
		{NvencPreset: "p1"},
		{NvencPreset: "p7"},
	}
	for _, job := range valid {
		if err := Validate(job); err != nil {
			t.Errorf("job %+v deveria ser válido: %v", job, err)
		}
	}

	invalid := []struct {
		name string
		job  Job
		want string
	}{
		{"corte invertido", Job{TrimStartSec: 20, TrimEndSec: 10}, "fim do corte"},
		{"corte igual", Job{TrimStartSec: 10, TrimEndSec: 10}, "fim do corte"},
		{"corte negativo", Job{TrimStartSec: -1}, "negativos"},
		{"fps negativo", Job{FPS: -1}, "FPS"},
		{"rotação inválida", Job{Rotate: 45}, "rotação"},
		{"escala inválida", Job{Scale: "hd"}, "escala"},
		{"preset nvenc inválido", Job{NvencPreset: "fast"}, "preset NVENC"},
		{"split + trim", Job{Split: &split.Spec{Parts: 2}, TrimStartSec: 5}, "não podem ser combinados"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.job)
			if err == nil {
				t.Fatalf("esperava erro para %+v", tc.job)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("erro %q deveria conter %q", err.Error(), tc.want)
			}
		})
	}
}

func TestTrimArgsBeforeInput(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	info := &media.Info{DurationSec: 600, HasAudio: true}
	p := presets.Preset{Mode: "size", SizeMB: 10, AudioBitrate: "96k", CRF: 23}
	job := Job{InputPath: "in.mp4", OutputPath: "out.mp4", TrimStartSec: 60, TrimEndSec: 180}
	seg := split.Segment{Index: 1, EndSec: 600}

	p1, p2, err := buildSizePasses(context.Background(), "j", job, p, info, seg)
	if err != nil {
		t.Fatalf("buildSizePasses: %v", err)
	}
	for name, args := range map[string][]string{"pass1": p1.Args, "pass2": p2.Args} {
		if !beforeInput(args, "-ss", "60.000") || !beforeInput(args, "-to", "180.000") {
			t.Errorf("%s: trim deveria ficar antes do -i, args: %v", name, args)
		}
	}

	budget, err := estimate.BudgetFor(p, info, 120)
	if err != nil {
		t.Fatalf("BudgetFor: %v", err)
	}
	if !containsStr(p2.Args, "-b:v", strconv.Itoa(budget.VideoBitrate)) {
		t.Errorf("orçamento deveria usar a duração do corte (120s), args: %v", p2.Args)
	}
}

func TestRemoveAudioArgs(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	job := Job{InputPath: "in.mp4", OutputPath: "out.mp4", RemoveAudio: true}
	info := &media.Info{DurationSec: 80, HasAudio: true}

	_, p2, err := buildSizePasses(context.Background(), "j", job, presets.Preset{Mode: "size", SizeMB: 10, AudioBitrate: "96k"}, info, split.Segment{Index: 1, EndSec: 80})
	if err != nil {
		t.Fatalf("buildSizePasses: %v", err)
	}
	if !containsStr(p2.Args, "-an") {
		t.Errorf("pass2 deveria incluir -an, args: %v", p2.Args)
	}
	if containsStr(p2.Args, "-c:a", "aac") {
		t.Errorf("pass2 não deveria reencodar áudio, args: %v", p2.Args)
	}

	crf := buildCrfPass(context.Background(), job, presets.Preset{CRF: 23, AudioBitrate: "96k"}, info, split.Segment{})
	if containsStr(crf.Args, "-c:a", "aac") {
		t.Errorf("CRF com remover áudio não deveria incluir -c:a, args: %v", crf.Args)
	}
}

func TestCustomScaleAndFPSArgs(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	job := Job{InputPath: "in.mp4", OutputPath: "out.mp4", Scale: "640x480", FPS: 24, Rotate: 90}
	cmd := buildCrfPass(context.Background(), job, presets.Preset{CRF: 23, AudioBitrate: "96k"}, &media.Info{DurationSec: 80, HasAudio: true}, split.Segment{})
	if !containsStr(cmd.Args, "-vf", "transpose=1,scale=640x480:force_original_aspect_ratio=decrease,fps=24.000") {
		t.Errorf("cadeia -vf inesperada, args: %v", cmd.Args)
	}
}

func TestOriginalScaleOmitsVF(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	job := Job{InputPath: "in.mp4", OutputPath: "out.mp4", Scale: ScaleOriginal}
	cmd := buildCrfPass(context.Background(), job, presets.Preset{CRF: 23, AudioBitrate: "96k"}, &media.Info{DurationSec: 80, HasAudio: true}, split.Segment{})
	if containsStr(cmd.Args, "-vf") {
		t.Errorf("escala original não deveria ter -vf, args: %v", cmd.Args)
	}
}

func TestBuildThumbnail(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ffmpeg.exe")
	if err := os.WriteFile(bin, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cmdutil.SetOverride("ffmpeg", bin)
	t.Cleanup(cmdutil.ClearOverrides)

	cmd, err := buildThumbnail(context.Background(), "out.mp4", "poster.png", 12.5)
	if err != nil {
		t.Fatalf("buildThumbnail: %v", err)
	}
	if !containsStr(cmd.Args, "-ss", "12.500") || !containsStr(cmd.Args, "-frames:v", "1") {
		t.Errorf("args de thumbnail inesperados: %v", cmd.Args)
	}
	if last := cmd.Args[len(cmd.Args)-1]; last != "poster.png" {
		t.Errorf("destino do thumbnail deveria ser o último arg, got %q", last)
	}
}

func TestRunGeneratesThumbnail(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "out.mp4")
	thumb := filepath.Join(dir, "poster.png")

	got := captureCompressEvents(t)
	Run(context.Background(), "jthumb", Job{
		InputPath:     "in.mp4",
		OutputPath:    out,
		PresetID:      "youtube",
		ThumbnailPath: thumb,
	})
	if e := findErr(got); e != "" {
		t.Fatalf("erro inesperado: %s", e)
	}
	if _, err := os.Stat(thumb); err != nil {
		t.Errorf("thumbnail não foi criada: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("saída não foi criada: %v", err)
	}
}

func TestRunRejectsInvalidAdjustments(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	got := captureCompressEvents(t)
	Run(context.Background(), "jbad", Job{
		InputPath:    "in.mp4",
		OutputPath:   filepath.Join(t.TempDir(), "out.mp4"),
		PresetID:     "youtube",
		TrimStartSec: 30,
		TrimEndSec:   10,
	})
	if e := findErr(got); !strings.Contains(e, "fim do corte") {
		t.Errorf("esperava erro de corte invertido, got %q", e)
	}
}

func TestNvencPresetArgs(t *testing.T) {
	skipOnWindows(t)
	fakeToolchain(t)
	info := &media.Info{DurationSec: 80, HasAudio: true}
	seg := split.Segment{Index: 1, EndSec: 80}

	fast := buildCrfPass(context.Background(), Job{InputPath: "in.mp4", OutputPath: "out.mp4", NvencPreset: "p1"}, presets.Preset{CRF: 23, AudioBitrate: "96k", Hardware: encode.HWNVENC}, info, seg)
	if !containsStr(fast.Args, "-preset", "p1") {
		t.Errorf("com preset p1 deveria incluir -preset p1, args: %v", fast.Args)
	}

	def := buildCrfPass(context.Background(), Job{InputPath: "in.mp4", OutputPath: "out.mp4"}, presets.Preset{CRF: 23, AudioBitrate: "96k", Hardware: encode.HWNVENC}, info, seg)
	if !containsStr(def.Args, "-preset", "p4") {
		t.Errorf("sem preset deveria usar o default p4, args: %v", def.Args)
	}
}

func beforeInput(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-i" {
			return false
		}
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}
