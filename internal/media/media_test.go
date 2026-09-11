package media

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// makeTestVideo generates a tiny 2s H.264/aac clip with ffmpeg.
func makeTestVideo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg não está instalado")
	}

	out := filepath.Join(t.TempDir(), "input.mp4")
	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "lavfi", "-i", "color=c=red:s=320x240:d=2",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono",
		"-t", "2",
		"-c:v", "libx264", "-preset", "ultrafast",
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

func TestProbe(t *testing.T) {
	input := makeTestVideo(t)

	info, err := Probe(input)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if !info.HasVideo {
		t.Error("expected video stream")
	}
	if !info.HasAudio {
		t.Error("expected audio stream")
	}
	if info.DurationSec < 1.5 || info.DurationSec > 3.0 {
		t.Errorf("duration = %v; want ~2s", info.DurationSec)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Errorf("size = %dx%d; want 320x240", info.Width, info.Height)
	}

	t.Run("missing file", func(t *testing.T) {
		if _, err := Probe(filepath.Join(t.TempDir(), "nope.mp4")); err == nil {
			t.Error("expected error for missing file")
		}
	})
}
