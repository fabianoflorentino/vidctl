package estimate_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/compress"
	"github.com/fabianoflorentino/vidctl/internal/estimate"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
)

// makeNoisyVideo generates a short clip with heavy temporal+spatial noise so
// the two-pass encoder actually spends its bitrate budget. Trivial content
// would undershoot the target and make the parity check meaningless.
func makeNoisyVideo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg não está instalado")
	}

	out := filepath.Join(t.TempDir(), "input.mp4")
	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "lavfi", "-i", "testsrc2=size=320x240:rate=30",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono",
		"-t", "4",
		"-vf", "noise=alls=24:allf=t+u",
		"-c:v", "libx264", "-preset", "ultrafast", "-crf", "18",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-movflags", "+faststart",
		out,
	)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("falha ao gerar vídeo de teste: %v\n%s", err, outBytes)
	}
	return out
}

// TestSizeMatchesCompressRun checks that the estimate of a size-limited job
// lands near what compress.Run actually produces on content that binds the
// bitrate budget.
func TestSizeMatchesCompressRun(t *testing.T) {
	input := makeNoisyVideo(t)
	out := filepath.Join(t.TempDir(), "out.mp4")

	info, err := media.Probe(input)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}

	job := compress.Job{
		InputPath:  input,
		OutputPath: out,
		PresetID:   "whatsapp-status",
		SizeMB:     1,
	}
	preset, ok := compress.EffectivePreset(job)
	if !ok {
		t.Fatal("preset não encontrado")
	}

	got, err := estimate.Size(info, preset, nil)
	if err != nil {
		t.Fatalf("estimate.Size: %v", err)
	}
	if !got.Available {
		t.Fatal("esperava estimativa disponível no modo size")
	}

	var failed bool
	events.SetEmitter(func(_ string, data any) {
		if _, ok := data.(events.ErrorEvent); ok {
			failed = true
		}
	})
	defer events.SetEmitter(nil)

	if err := compress.Run(t.Context(), "parity", job); err != nil {
		t.Fatalf("compress.Run: %v", err)
	}
	if failed {
		t.Fatal("job falhou ao comprimir")
	}

	st, err := os.Stat(out)
	if err != nil {
		t.Fatalf("arquivo de saída ausente: %v", err)
	}
	actualMB := float64(st.Size()) / (1024 * 1024)

	// O áudio silencioso real gasta bem menos que os 96k reservados no
	// orçamento e o x264 flutua um pouco em volta do alvo: faixa larga.
	t.Logf("estimativa %.3f MB; saída real %.3f MB", got.TargetSizeMB, actualMB)
	if actualMB < got.TargetSizeMB*0.80 || actualMB > got.TargetSizeMB*1.10 {
		t.Errorf("saída real %.3f MB fugiu da estimativa %.3f MB", actualMB, got.TargetSizeMB)
	}
}
