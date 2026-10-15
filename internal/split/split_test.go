package split

import (
	"math"
	"strings"
	"testing"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

func checkSegments(t *testing.T, got, want []Segment) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for i, seg := range got {
		w := want[i]
		if seg.Index != w.Index || !approxEqual(seg.StartSec, w.StartSec) || !approxEqual(seg.EndSec, w.EndSec) {
			t.Errorf("seg[%d] = %+v, want %+v", i, seg, w)
		}
	}
}

func TestPlanByParts(t *testing.T) {
	tests := []struct {
		name        string
		durationSec float64
		spec        Spec
		wantSegs    []Segment
		wantErr     string
	}{
		{
			name:        "30min em 6 partes de 5min",
			durationSec: 1800,
			spec:        Spec{Parts: 6},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 300},
				{Index: 2, StartSec: 300, EndSec: 600},
				{Index: 3, StartSec: 600, EndSec: 900},
				{Index: 4, StartSec: 900, EndSec: 1200},
				{Index: 5, StartSec: 1200, EndSec: 1500},
				{Index: 6, StartSec: 1500, EndSec: 1800},
			},
		},
		{
			name:        "32min em 6 partes iguais (5m20s cada)",
			durationSec: 1920,
			spec:        Spec{Parts: 6},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 320},
				{Index: 2, StartSec: 320, EndSec: 640},
				{Index: 3, StartSec: 640, EndSec: 960},
				{Index: 4, StartSec: 960, EndSec: 1280},
				{Index: 5, StartSec: 1280, EndSec: 1600},
				{Index: 6, StartSec: 1600, EndSec: 1920},
			},
		},
		{
			name:        "partes curtas sao permitidas",
			durationSec: 300,
			spec:        Spec{Parts: 6},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 50},
				{Index: 2, StartSec: 50, EndSec: 100},
				{Index: 3, StartSec: 100, EndSec: 150},
				{Index: 4, StartSec: 150, EndSec: 200},
				{Index: 5, StartSec: 200, EndSec: 250},
				{Index: 6, StartSec: 250, EndSec: 300},
			},
		},
		{name: "uma parte nao e corte", durationSec: 1800, spec: Spec{Parts: 1}, wantErr: "pelo menos 2 partes"},
		{name: "partes demais", durationSec: 1800, spec: Spec{Parts: MaxParts + 1}, wantErr: "maximum"},
		{
			name:        "video de 30s cortado em 2",
			durationSec: 30,
			spec:        Spec{Parts: 2},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 15},
				{Index: 2, StartSec: 15, EndSec: 30},
			},
		},
		{
			name:        "1:49 cortado em 2 partes",
			durationSec: 109,
			spec:        Spec{Parts: 2},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 54.5},
				{Index: 2, StartSec: 54.5, EndSec: 109},
			},
		},
		{name: "duracao invalida", durationSec: 0, spec: Spec{Parts: 2}, wantErr: "invalid video duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Plan(tt.durationSec, tt.spec)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want contendo %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			checkSegments(t, got, tt.wantSegs)
		})
	}
}

func TestPlanByMinutes(t *testing.T) {
	tests := []struct {
		name        string
		durationSec float64
		spec        Spec
		wantSegs    []Segment
		wantErr     string
	}{
		{
			name:        "30min de 5 em 5min",
			durationSec: 1800,
			spec:        Spec{MinutesEach: 5},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 300},
				{Index: 2, StartSec: 300, EndSec: 600},
				{Index: 3, StartSec: 600, EndSec: 900},
				{Index: 4, StartSec: 900, EndSec: 1200},
				{Index: 5, StartSec: 1200, EndSec: 1500},
				{Index: 6, StartSec: 1500, EndSec: 1800},
			},
		},
		{
			name:        "sobra menor que 1min vira a ultima parte",
			durationSec: 1830, // 6x5min = 1800, sobra 30s -> parte 7 de 30s
			spec:        Spec{MinutesEach: 5},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 300},
				{Index: 2, StartSec: 300, EndSec: 600},
				{Index: 3, StartSec: 600, EndSec: 900},
				{Index: 4, StartSec: 900, EndSec: 1200},
				{Index: 5, StartSec: 1200, EndSec: 1500},
				{Index: 6, StartSec: 1500, EndSec: 1800},
				{Index: 7, StartSec: 1800, EndSec: 1830},
			},
		},
		{
			name:        "1:49 com 1min por parte vira 1:00 + 0:49",
			durationSec: 109,
			spec:        Spec{MinutesEach: 1},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 60},
				{Index: 2, StartSec: 60, EndSec: 109},
			},
		},
		{
			name:        "sobra de 2min vira parte final menor",
			durationSec: 1920, // 6x5min + 120s >= 60 -> parte extra
			spec:        Spec{MinutesEach: 5},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 300},
				{Index: 2, StartSec: 300, EndSec: 600},
				{Index: 3, StartSec: 600, EndSec: 900},
				{Index: 4, StartSec: 900, EndSec: 1200},
				{Index: 5, StartSec: 1200, EndSec: 1500},
				{Index: 6, StartSec: 1500, EndSec: 1800},
				{Index: 7, StartSec: 1800, EndSec: 1920},
			},
		},
		{
			name:        "video cabe em uma parte nao gera corte",
			durationSec: 240,
			spec:        Spec{MinutesEach: 5},
			wantErr:     "fits in 1 part",
		},
		{
			name:        "minutos fracionarios cortam em partes de 1m30s",
			durationSec: 720, // 8x90s = 720
			spec:        Spec{MinutesEach: 1.5},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 90},
				{Index: 2, StartSec: 90, EndSec: 180},
				{Index: 3, StartSec: 180, EndSec: 270},
				{Index: 4, StartSec: 270, EndSec: 360},
				{Index: 5, StartSec: 360, EndSec: 450},
				{Index: 6, StartSec: 450, EndSec: 540},
				{Index: 7, StartSec: 540, EndSec: 630},
				{Index: 8, StartSec: 630, EndSec: 720},
			},
		},
		{
			name:        "escolhas conflitantes",
			durationSec: 1800,
			spec:        Spec{Parts: 6, MinutesEach: 5},
			wantErr:     "only one",
		},
		{
			name:        "nenhuma escolha",
			durationSec: 1800,
			spec:        Spec{},
			wantErr:     "requires a number of parts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Plan(tt.durationSec, tt.spec)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want contendo %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			checkSegments(t, got, tt.wantSegs)
		})
	}
}

func TestPlanByMinutesMaxParts(t *testing.T) {
	_, err := Plan(61*60, Spec{MinutesEach: 1})
	if err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("err = %v, want maximum-parts error", err)
	}
}

func TestFilename(t *testing.T) {
	tests := []struct {
		path  string
		index int
		total int
		want  string
	}{
		{"a.mp4", 3, 6, "a-part3.mp4"},
		{"/tmp/meu_video.mkv", 1, 12, "/tmp/meu_video-part01.mkv"},
		{"a.mov", 10, 12, "a-part10.mov"},
		{"sem-ext", 2, 3, "sem-ext-part2"},
	}
	for _, tt := range tests {
		if got := Filename(tt.path, tt.index, tt.total); got != tt.want {
			t.Errorf("Filename(%q,%d,%d) = %q, want %q", tt.path, tt.index, tt.total, got, tt.want)
		}
	}
}

func TestSegmentDuration(t *testing.T) {
	if got := (Segment{StartSec: 12.5, EndSec: 60}).Duration(); !approxEqual(got, 47.5) {
		t.Errorf("Duration() = %v, want 47.5", got)
	}
}
