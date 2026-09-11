package compress

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/events"
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

func TestRunSizeMode(t *testing.T) {
	input := makeTestVideo(t)
	out := filepath.Join(t.TempDir(), "out.mp4")

	type capture struct {
		name string
		data any
	}
	got := make([]capture, 0, 4)
	events.SetEmitter(func(name string, data any) {
		got = append(got, capture{name: name, data: data})
	})
	defer events.SetEmitter(nil)

	job := Job{
		InputPath:  input,
		OutputPath: out,
		PresetID:   "whatsapp-status",
		SizeMB:     2,
	}
	Run(t.Context(), "test-job", job)

	var done *events.DoneEvent
	var errEvt *events.ErrorEvent
	for _, c := range got {
		switch ev := c.data.(type) {
		case events.DoneEvent:
			done = &ev
		case events.ErrorEvent:
			errEvt = &ev
		}
	}

	if errEvt != nil {
		t.Fatalf("job failed: %s", errEvt.Error)
	}
	if done == nil {
		t.Fatal("expected done event")
	}
	if done.OutputPath != out {
		t.Errorf("done output = %q; want %q", done.OutputPath, out)
	}
	if done.SizeBytes == 0 {
		t.Error("expected output file to have bytes")
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("output file missing: %v", err)
	}
}

func TestRunCrfMode(t *testing.T) {
	input := makeTestVideo(t)
	out := filepath.Join(t.TempDir(), "out.mp4")

	var done bool
	var errEvt *events.ErrorEvent
	events.SetEmitter(func(name string, data any) {
		if err, ok := data.(events.ErrorEvent); ok {
			errEvt = &err
		}
		if _, ok := data.(events.DoneEvent); ok {
			done = true
		}
	})
	defer events.SetEmitter(nil)

	job := Job{
		InputPath:  input,
		OutputPath: out,
		PresetID:   "youtube",
	}
	Run(t.Context(), "test-job-crf", job)

	if errEvt != nil {
		t.Fatalf("job failed: %s", errEvt.Error)
	}
	if !done {
		t.Fatal("expected done event in CRF mode")
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("output file missing: %v", err)
	}
}

func TestRunValidation(t *testing.T) {
	tests := []struct {
		name     string
		job      Job
		wantFail bool
	}{
		{
			name: "same input and output",
			job: Job{
				InputPath:  "/tmp/a.mp4",
				OutputPath: "/tmp/a.mp4",
				PresetID:   "whatsapp-status",
			},
			wantFail: true,
		},
		{
			name: "unknown preset",
			job: Job{
				InputPath:  "/tmp/a.mp4",
				OutputPath: "/tmp/b.mp4",
				PresetID:   "nao-existe",
			},
			wantFail: true,
		},
		{
			name: "empty input",
			job: Job{
				InputPath:  "",
				OutputPath: "/tmp/b.mp4",
				PresetID:   "whatsapp-status",
			},
			wantFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var failed bool
			events.SetEmitter(func(_ string, data any) {
				if _, ok := data.(events.ErrorEvent); ok {
					failed = true
				}
			})
			defer events.SetEmitter(nil)

			Run(t.Context(), "job-"+tt.name, tt.job)
			if failed != tt.wantFail {
				t.Errorf("failed = %v; want %v", failed, tt.wantFail)
			}
		})
	}
}
