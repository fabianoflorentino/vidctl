package compress

import (
	"math"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
)

func TestBitrateToBits(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want float64
	}{
		{name: "kilobit", in: "96k", want: 96_000},
		{name: "megabit", in: "1M", want: 1_000_000},
		{name: "raw", in: "500000", want: 500_000},
		{name: "empty", in: "", want: 0},
		{name: "invalid", in: "abc", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bitrateToBits(tt.in)
			if got != tt.want {
				t.Errorf("bitrateToBits(%q) = %v; want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestComputeSizeBudget(t *testing.T) {
	preset := presets.Preset{Mode: "size", SizeMB: 10, AudioBitrate: "96k"}

	tests := []struct {
		name     string
		duration float64
		hasAudio bool
		wantErr  bool
		wantNear int
	}{
		{
			name:     "80s with audio at 10MB",
			duration: 80,
			hasAudio: true,
			wantNear: 900_000,
		},
		{
			name:     "80s without audio gets more video budget",
			duration: 80,
			hasAudio: false,
			wantNear: 996_000,
		},
		{
			name:     "target too small for long video",
			duration: 3600,
			hasAudio: true,
			wantErr:  true,
		},
		{
			name:     "zero duration",
			duration: 0,
			hasAudio: true,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &media.Info{
				DurationSec: tt.duration,
				HasAudio:    tt.hasAudio,
			}
			lines, err := computeSizeBudget(Job{}, preset, info)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Allow generous tolerance since exact bitrate math may shift.
			if math.Abs(float64(lines.videoBitrate)-float64(tt.wantNear)) > 50_000 {
				t.Errorf("videoBitrate = %d; want ~%d", lines.videoBitrate, tt.wantNear)
			}
			if lines.maxRate < lines.videoBitrate {
				t.Errorf("maxRate %d must be >= bitrate %d", lines.maxRate, lines.videoBitrate)
			}
		})
	}
}

func TestComputeSizeBudgetFloorsBitrate(t *testing.T) {
	preset := presets.Preset{Mode: "size", SizeMB: 1, AudioBitrate: "64k"}
	info := &media.Info{DurationSec: 2000, HasAudio: false}
	lines, err := computeSizeBudget(Job{}, preset, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lines.videoBitrate != 50_000 {
		t.Errorf("bitrate = %d; want floor 50000", lines.videoBitrate)
	}
}
