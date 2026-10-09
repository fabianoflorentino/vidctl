package estimate

import (
	"math"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/split"
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
			if got := BitrateToBits(tt.in); got != tt.want {
				t.Errorf("BitrateToBits(%q) = %v; want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestBudgetFor(t *testing.T) {
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
			info := &media.Info{DurationSec: tt.duration, HasAudio: tt.hasAudio}
			budget, err := BudgetFor(preset, info, tt.duration)
			if tt.wantErr {
				if err == nil {
					t.Fatal("esperava erro, obtive nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if math.Abs(float64(budget.VideoBitrate)-float64(tt.wantNear)) > 50_000 {
				t.Errorf("VideoBitrate = %d; queria ~%d", budget.VideoBitrate, tt.wantNear)
			}
			if budget.MaxRate < budget.VideoBitrate {
				t.Errorf("MaxRate %d deve ser >= VideoBitrate %d", budget.MaxRate, budget.VideoBitrate)
			}
			if budget.BufSize != budget.VideoBitrate*2 {
				t.Errorf("BufSize = %d; queria %d", budget.BufSize, budget.VideoBitrate*2)
			}
		})
	}
}

func TestBudgetForFloorsBitrate(t *testing.T) {
	preset := presets.Preset{Mode: "size", SizeMB: 1, AudioBitrate: "64k"}
	info := &media.Info{DurationSec: 2000, HasAudio: false}
	budget, err := BudgetFor(preset, info, 2000)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if budget.VideoBitrate != 50_000 {
		t.Errorf("bitrate = %d; queria o piso 50000", budget.VideoBitrate)
	}
}

func TestSize(t *testing.T) {
	sizePreset := presets.Preset{Mode: "size", SizeMB: 10, AudioBitrate: "96k"}
	tests := []struct {
		name          string
		preset        presets.Preset
		info          *media.Info
		spec          *split.Spec
		wantErr       bool
		wantAvail     bool
		wantTargetMB  float64
		wantSavedPct  float64
		wantVideoKbps int
		wantAudioKbps int
		wantCurrentMB float64
	}{
		{
			name:          "modo size exato com áudio",
			preset:        sizePreset,
			info:          &media.Info{DurationSec: 80, SizeMB: 26, HasAudio: true},
			wantAvail:     true,
			wantTargetMB:  9.5,
			wantSavedPct:  100 * (1 - 9.5/26),
			wantVideoKbps: 900,
			wantAudioKbps: 96,
			wantCurrentMB: 26,
		},
		{
			name:          "sem áudio o orçamento todo vira vídeo",
			preset:        sizePreset,
			info:          &media.Info{DurationSec: 80, SizeMB: 26, HasAudio: false},
			wantAvail:     true,
			wantTargetMB:  9.5,
			wantSavedPct:  100 * (1 - 9.5/26),
			wantVideoKbps: 996,
			wantAudioKbps: 0,
			wantCurrentMB: 26,
		},
		{
			name:          "piso de bitrate aumenta a previsão",
			preset:        presets.Preset{Mode: "size", SizeMB: 1, AudioBitrate: "64k"},
			info:          &media.Info{DurationSec: 2000, SizeMB: 300, HasAudio: false},
			wantAvail:     true,
			wantTargetMB:  11.9209,
			wantSavedPct:  100 * (1 - 11.9209/300),
			wantVideoKbps: 50,
			wantAudioKbps: 0,
			wantCurrentMB: 300,
		},
		{
			name:      "alvo impossível devolve erro",
			preset:    presets.Preset{Mode: "size", SizeMB: 1, AudioBitrate: "96k"},
			info:      &media.Info{DurationSec: 3600, HasAudio: true},
			wantErr:   true,
			wantAvail: false,
		},
		{
			name:      "duração inválida devolve erro",
			preset:    sizePreset,
			info:      &media.Info{DurationSec: 0, HasAudio: true},
			wantErr:   true,
			wantAvail: false,
		},
		{
			name:          "modo crf não tem estimativa determinística",
			preset:        presets.Preset{Mode: "crf", CRF: 23, AudioBitrate: "128k"},
			info:          &media.Info{DurationSec: 80, SizeMB: 26, HasAudio: true},
			wantAvail:     false,
			wantVideoKbps: 0,
			wantAudioKbps: 128,
			wantCurrentMB: 26,
		},
		{
			name:          "corte em partes soma um alvo por parte",
			preset:        sizePreset,
			info:          &media.Info{DurationSec: 160, SizeMB: 26, HasAudio: true},
			spec:          &split.Spec{Parts: 2},
			wantAvail:     true,
			wantTargetMB:  19.0,
			wantSavedPct:  100 * (1 - 19.0/26),
			wantVideoKbps: 900,
			wantAudioKbps: 96,
			wantCurrentMB: 26,
		},
		{
			name:      "split inválido devolve erro",
			preset:    sizePreset,
			info:      &media.Info{DurationSec: 80, SizeMB: 26, HasAudio: true},
			spec:      &split.Spec{},
			wantErr:   true,
			wantAvail: false,
		},
		{
			name:          "sem tamanho atual a economia fica em zero",
			preset:        sizePreset,
			info:          &media.Info{DurationSec: 80, SizeMB: 0, HasAudio: true},
			wantAvail:     true,
			wantTargetMB:  9.5,
			wantSavedPct:  0,
			wantVideoKbps: 900,
			wantAudioKbps: 96,
			wantCurrentMB: 0,
		},
		{
			name:          "previsão maior que o original devolve economia negativa",
			preset:        presets.Preset{Mode: "size", SizeMB: 10, AudioBitrate: "96k"},
			info:          &media.Info{DurationSec: 80, SizeMB: 5, HasAudio: true},
			wantAvail:     true,
			wantTargetMB:  9.5,
			wantSavedPct:  100 * (1 - 9.5/5),
			wantVideoKbps: 900,
			wantAudioKbps: 96,
			wantCurrentMB: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Size(tt.info, tt.preset, tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatal("esperava erro, obtive nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if res.Available != tt.wantAvail {
				t.Errorf("Available = %v; queria %v", res.Available, tt.wantAvail)
			}
			if res.Mode != tt.preset.Mode {
				t.Errorf("Mode = %q; queria %q", res.Mode, tt.preset.Mode)
			}
			if math.Abs(res.TargetSizeMB-tt.wantTargetMB) > 0.01 {
				t.Errorf("TargetSizeMB = %.4f; queria %.4f", res.TargetSizeMB, tt.wantTargetMB)
			}
			if math.Abs(res.SavedPct-tt.wantSavedPct) > 0.1 {
				t.Errorf("SavedPct = %.3f; queria %.3f", res.SavedPct, tt.wantSavedPct)
			}
			if res.VideoKbps != tt.wantVideoKbps {
				t.Errorf("VideoKbps = %d; queria %d", res.VideoKbps, tt.wantVideoKbps)
			}
			if res.AudioKbps != tt.wantAudioKbps {
				t.Errorf("AudioKbps = %d; queria %d", res.AudioKbps, tt.wantAudioKbps)
			}
			if res.CurrentSizeMB != tt.wantCurrentMB {
				t.Errorf("CurrentSizeMB = %f; queria %f", res.CurrentSizeMB, tt.wantCurrentMB)
			}
		})
	}
}

func TestSizeWithoutInfo(t *testing.T) {
	if _, err := Size(nil, presets.Preset{Mode: "size", SizeMB: 10}, nil); err == nil {
		t.Error("esperava erro sem info do vídeo")
	}
}
