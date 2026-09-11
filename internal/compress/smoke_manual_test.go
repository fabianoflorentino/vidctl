package compress

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
)

// Manual smoke test against a real file. Run only when VIDCTL_SMOKE is set:
//
//	VIDCTL_SMOKE="/path/video.mp4" go test -run TestSmokeReal -v ./internal/compress/
func TestSmokeReal(t *testing.T) {
	input := os.Getenv("VIDCTL_SMOKE")
	if input == "" {
		t.Skip("VIDCTL_SMOKE not set")
	}

	info, err := media.Probe(input)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	t.Logf("duration=%.1fs size=%dx%d hasAudio=%v sizeMB=%.1f", info.DurationSec, info.Width, info.Height, info.HasAudio, info.SizeMB)

	out := filepath.Join(t.TempDir(), "comprimido.mp4")

	var doneMB float64
	events.SetEmitter(func(_ string, data any) {
		if d, ok := data.(events.DoneEvent); ok {
			doneMB = d.SizeMB
		}
		if e, ok := data.(events.ErrorEvent); ok {
			t.Errorf("compress error: %s", e.Error)
		}
	})
	defer events.SetEmitter(nil)

	job := Job{
		InputPath:  input,
		OutputPath: out,
		PresetID:   "whatsapp-status",
		SizeMB:     10,
	}
	Run(t.Context(), "smoke", job)

	if doneMB == 0 {
		t.Fatal("job did not complete")
	}
	t.Logf("resultado: %.2f MB", doneMB)
	if doneMB > 10.2 {
		t.Errorf("resultado %.2f MB estourou o alvo de 10 MB", doneMB)
	}
}
