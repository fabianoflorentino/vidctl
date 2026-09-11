package media

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell bins não executam no Windows")
	}
}

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

func fakeBin(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, name)
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func probeScript(payload string) string {
	return "#!/bin/sh\nprintf '%s\\n' '" + payload + "'\n"
}

func probeInput(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "input.mp4")
	if err := os.WriteFile(p, make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProbeFFprobeMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := Probe("/tmp/x.mp4"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProbeWithFakeFFprobe(t *testing.T) {
	skipOnWindows(t)
	in := probeInput(t)
	dir := fakeBin(t, "ffprobe", probeScript(
		`{"streams":[
			{"codec_type":"video","codec_name":"h264","width":320,"height":240,"duration":2.0},
			{"codec_type":"audio","codec_name":"aac"}
		],"format":{"duration":2.0}}`))
	t.Setenv("PATH", dir)

	info, err := Probe(in)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if !info.HasVideo || info.VideoCodec != "h264" {
		t.Errorf("video inesperado: %+v", info)
	}
	if !info.HasAudio || info.AudioCodec != "aac" {
		t.Errorf("audio inesperado: %+v", info)
	}
	if info.DurationSec != 2.0 {
		t.Errorf("duration = %v; want 2", info.DurationSec)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Errorf("size = %dx%d; want 320x240", info.Width, info.Height)
	}
	if info.SizeMB <= 0 {
		t.Errorf("SizeMB deve ser > 0, got %v", info.SizeMB)
	}
}

func TestProbeRotationSwapsDimensions(t *testing.T) {
	skipOnWindows(t)
	in := probeInput(t)
	dir := fakeBin(t, "ffprobe", probeScript(
		`{"streams":[
			{"codec_type":"video","codec_name":"h264","width":320,"height":240,
			 "duration":2.0,"tags":{"rotate":"90"}}
		],"format":{"duration":2.0}}`))
	t.Setenv("PATH", dir)

	info, err := Probe(in)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if info.Width != 240 || info.Height != 320 {
		t.Errorf("rotação 90 deve trocar dimensões, got %dx%d", info.Width, info.Height)
	}
}

func TestProbeDurationFallbackFromStream(t *testing.T) {
	skipOnWindows(t)
	in := probeInput(t)
	dir := fakeBin(t, "ffprobe", probeScript(
		`{"streams":[
			{"codec_type":"video","codec_name":"h264","width":320,"height":240,"duration":"1.5"}
		],"format":{"duration":0}}`))
	t.Setenv("PATH", dir)

	info, err := Probe(in)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if info.DurationSec != 1.5 {
		t.Errorf("duration = %v; want 1.5 (fallback do stream)", info.DurationSec)
	}
}

func TestProbeNoVideoStream(t *testing.T) {
	skipOnWindows(t)
	in := probeInput(t)
	dir := fakeBin(t, "ffprobe", probeScript(
		`{"streams":[{"codec_type":"audio","codec_name":"aac"}],"format":{"duration":2.0}}`))
	t.Setenv("PATH", dir)

	_, err := Probe(in)
	if err == nil || !strings.Contains(err.Error(), "não contém um stream de vídeo") {
		t.Fatalf("esperava erro de stream de vídeo, got %v", err)
	}
}

func TestProbeZeroDuration(t *testing.T) {
	skipOnWindows(t)
	in := probeInput(t)
	dir := fakeBin(t, "ffprobe", probeScript(
		`{"streams":[{"codec_type":"video","codec_name":"h264","width":1,"height":1,"duration":0}],"format":{"duration":0}}`))
	t.Setenv("PATH", dir)

	_, err := Probe(in)
	if err == nil || !strings.Contains(err.Error(), "duração") {
		t.Fatalf("esperava erro de duração, got %v", err)
	}
}

func TestProbeFFprobeFailure(t *testing.T) {
	skipOnWindows(t)
	dir := fakeBin(t, "ffprobe", "#!/bin/sh\nexit 1\n")
	t.Setenv("PATH", dir)

	if _, err := Probe("/tmp/x.mp4"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProbeInvalidJSON(t *testing.T) {
	skipOnWindows(t)
	dir := fakeBin(t, "ffprobe", "#!/bin/sh\necho 'nope'\n")
	t.Setenv("PATH", dir)

	if _, err := Probe("/tmp/x.mp4"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFlexFloatUnmarshal(t *testing.T) {
	var f flexFloat
	if err := f.UnmarshalJSON([]byte(`"3.5"`)); err != nil || float64(f) != 3.5 {
		t.Errorf("string ok? f=%v err=%v", f, err)
	}
	if err := f.UnmarshalJSON([]byte(`42`)); err != nil || float64(f) != 42 {
		t.Errorf("numero ok? f=%v err=%v", f, err)
	}
	if err := f.UnmarshalJSON([]byte(`"oops"`)); err == nil {
		t.Error("expected error for invalid string")
	}
}
