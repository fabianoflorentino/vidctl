package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestClampPct(t *testing.T) {
	tests := []struct{ in, want float64 }{
		{-5, 0}, {0, 0}, {42.5, 42.5}, {100, 100}, {700, 100},
	}
	for _, tt := range tests {
		if got := clampPct(tt.in); got != tt.want {
			t.Errorf("clampPct(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseNvidia(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  float64
		found bool
	}{
		{"single", "42\n", 42, true},
		{"multi max", "30\n55\n12\n", 55, true},
		{"com percent", " 40 %\n", 40, true},
		{"lixo", "error: no driver\n", 0, false},
		{"vazio", "", 0, false},
		{"acima de 100", "110", 100, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseNvidia([]byte(tt.in))
			if ok != tt.found || got != tt.want {
				t.Errorf("parseNvidia(%q) = (%v,%v), want (%v,%v)", tt.in, got, ok, tt.want, tt.found)
			}
		})
	}
}

func TestIsFFmpeg(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"ffmpeg", true},
		{"ffmpeg.exe", true},
		{"/usr/bin/FFmpeg", true},
		{"ffprobe", false},
		{"chrome", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isFFmpeg(tt.name); got != tt.want {
			t.Errorf("isFFmpeg(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestFFmpegCPUPercentNormalized(t *testing.T) {
	// não há ffmpeg rodando neste teste: resultado deve ser 0 e dentro do intervalo
	got := ffmpegCPUPercent(t.Context())
	if got < 0 || got > 100 {
		t.Errorf("ffmpegCPUPercent = %v, fora de 0..100", got)
	}
}

func TestSnapshotSmoke(t *testing.T) {
	c := NewCollector()
	s := c.Snapshot()
	if s.MemTotalMB <= 0 {
		t.Errorf("MemTotalMB = %v, esperava > 0", s.MemTotalMB)
	}
	if s.CPU < 0 || s.CPU > 100 {
		t.Errorf("CPU = %v, fora de 0..100", s.CPU)
	}
	if s.GPU < -1 || s.GPU > 100 {
		t.Errorf("GPU = %v, esperava -1 ou 0..100", s.GPU)
	}
	if runtime.NumCPU() < 1 {
		t.Fatal("NumCPU inválido")
	}
}

func TestSnapshotWithFakeNvidia(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell bin não executam no Windows")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "nvidia-smi")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 37\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	c := NewCollector()
	s := c.Snapshot()
	if s.GPU != 37 {
		t.Errorf("GPU = %v, want 37", s.GPU)
	}
}

func TestSnapshotNvidiaMissingCached(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("LookPath no Windows exige .exe; PATH vazio não simula o caso")
	}
	t.Setenv("PATH", t.TempDir())
	c := NewCollector()
	s := c.Snapshot()
	if s.GPU != -1 {
		t.Fatalf("GPU = %v, want -1 sem nvidia-smi", s.GPU)
	}
	if !c.gpuProbed || c.gpuOK {
		t.Errorf("esperava cache de indisponibilidade, got probed=%v ok=%v", c.gpuProbed, c.gpuOK)
	}
	// segundo Snapshot usa o cache e não retorna GPU
	s2 := c.Snapshot()
	if s2.GPU != -1 {
		t.Errorf("GPU cacheado = %v, want -1", s2.GPU)
	}
}
