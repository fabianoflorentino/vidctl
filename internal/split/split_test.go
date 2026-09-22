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
			name:        "partes menores que o minimo sao rejeitadas",
			durationSec: 300,
			spec:        Spec{Parts: 6},
			wantErr:     "o mínimo é 1 min",
		},
		{name: "uma parte nao e corte", durationSec: 1800, spec: Spec{Parts: 1}, wantErr: "pelo menos 2 partes"},
		{name: "partes demais", durationSec: 1800, spec: Spec{Parts: MaxParts + 1}, wantErr: "máximo"},
		{name: "video curto demais", durationSec: 30, spec: Spec{Parts: 2}, wantErr: "mais curto que o mínimo"},
		{name: "duracao invalida", durationSec: 0, spec: Spec{Parts: 2}, wantErr: "duração do vídeo inválida"},
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
			name:        "sobra menor que 1min junta na ultima parte",
			durationSec: 1830, // 6x5min = 1800, sobra 30s (<60) -> parte 6 fica 1500-1830
			spec:        Spec{MinutesEach: 5},
			wantSegs: []Segment{
				{Index: 1, StartSec: 0, EndSec: 300},
				{Index: 2, StartSec: 300, EndSec: 600},
				{Index: 3, StartSec: 600, EndSec: 900},
				{Index: 4, StartSec: 900, EndSec: 1200},
				{Index: 5, StartSec: 1200, EndSec: 1500},
				{Index: 6, StartSec: 1500, EndSec: 1830},
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
			wantErr:     "cabe em 1 parte",
		},
		{
			name:        "escolhas conflitantes",
			durationSec: 1800,
			spec:        Spec{Parts: 6, MinutesEach: 5},
			wantErr:     "apenas uma opção",
		},
		{
			name:        "nenhuma escolha",
			durationSec: 1800,
			spec:        Spec{},
			wantErr:     "requer número de partes",
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
	if err == nil || !strings.Contains(err.Error(), "máximo") {
		t.Fatalf("err = %v, want erro de máximo de partes", err)
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
