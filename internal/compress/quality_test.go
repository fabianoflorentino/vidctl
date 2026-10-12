package compress

import (
	"strings"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/split"
)

func TestMinKbpsForHeight(t *testing.T) {
	tests := []struct{ height, want int }{
		{1920, 2000}, {1080, 2000}, {960, 1200}, {720, 1200}, {540, 700}, {480, 700}, {240, 400}, {0, 400},
	}
	for _, tt := range tests {
		if got := minKbpsForHeight(tt.height); got != tt.want {
			t.Errorf("minKbpsForHeight(%d) = %d, want %d", tt.height, got, tt.want)
		}
	}
}

func whatsappPreset() presets.Preset {
	return presets.Preset{Mode: "size", SizeMB: 10, AudioBitrate: "96k"}
}

func TestAdviseCrfIsAlwaysOK(t *testing.T) {
	info := &media.Info{DurationSec: 1800, Height: 1080, HasAudio: true}
	a := Advise(info, presets.Preset{Mode: "crf", CRF: 23, AudioBitrate: "96k"}, Job{})
	if !a.OK || a.Kbps != 0 {
		t.Errorf("CRF should not advise: %+v", a)
	}
}

func TestAdviseGoodClipNoWarning(t *testing.T) {
	// 60s 9:16 a 10 MB ≈ 1.23 Mbps ≥ 1200 kbps
	info := &media.Info{DurationSec: 60, Height: 960, HasAudio: true}
	a := Advise(info, whatsappPreset(), Job{})
	if !a.OK {
		t.Errorf("clipe curto deveria estar ok: %+v", a)
	}
	if a.Kbps < 1200 || a.MinKbps != 1200 {
		t.Errorf("kbps=%d min=%d, esperava kbps≥1200 min=1200", a.Kbps, a.MinKbps)
	}
}

func TestAdviseLongNoSplitSuggestsCut(t *testing.T) {
	// the user case: a long video (30min, 960p) at the 10 MB target with no cut
	info := &media.Info{DurationSec: 1800, Height: 960, HasAudio: true}
	a := Advise(info, whatsappPreset(), Job{})
	if a.OK {
		t.Fatal("esperava aviso")
	}
	if a.Suggestion.MinutesEach < 1 {
		t.Fatalf("expected a per-part minutes suggestion, got %+v", a.Suggestion)
	}
	if a.Suggestion.Parts < 2 {
		t.Errorf("esperava parts ≥ 2, got %+v", a.Suggestion)
	}
	if !strings.Contains(a.Message, "cut into parts") {
		t.Errorf("mensagem deveria sugerir corte: %q", a.Message)
	}
}

func TestAdviseSplitPartsSuggestsBiggerTarget(t *testing.T) {
	// 6 partes de 5min a 10MB cada ≈ 169 kbps → baixo; alvo por parte resolve
	info := &media.Info{DurationSec: 1800, Height: 960, HasAudio: true}
	job := Job{Split: &split.Spec{Parts: 6}}
	a := Advise(info, whatsappPreset(), job)
	if a.OK {
		t.Fatal("esperava aviso")
	}
	if a.Suggestion.SizeMB < 40 || a.Suggestion.SizeMB > maxTargetMB {
		t.Fatalf("SizeMB sugerido = %v, esperava ~49", a.Suggestion.SizeMB)
	}
	if !strings.Contains(a.Message, "raise the target") || !strings.Contains(a.Message, "for each part of 5min") {
		t.Errorf("mensagem inesperada: %q", a.Message)
	}
}

func TestAdviseSmallResolutionSuggestsSize(t *testing.T) {
	// 3min at 240p: ~67 kbps short → raise the target (no cut)
	info := &media.Info{DurationSec: 180, Height: 240, HasAudio: true}
	job := Job{}
	preset := presets.Preset{Mode: "size", SizeMB: 1, AudioBitrate: "64k"}
	a := Advise(info, preset, job)
	if a.OK {
		t.Fatal("esperava aviso")
	}
	if a.Suggestion.SizeMB <= 1 || a.Suggestion.MinutesEach != 0 {
		t.Errorf("expected a size-only suggestion, got %+v", a.Suggestion)
	}
}

func TestAdviseInvalidSplitSilent(t *testing.T) {
	info := &media.Info{DurationSec: 1800, Height: 960, HasAudio: true}
	a := Advise(info, whatsappPreset(), Job{Split: &split.Spec{Parts: 1}})
	if !a.OK {
		t.Errorf("invalid split is handled by the cut UI; advise should stay silent: %+v", a)
	}
}

func TestAdviseNoAudioUsesVideoOnly(t *testing.T) {
	// without audio the video budget is bigger, but 10 MB over 300s still yields only ~266 kbps < 400
	info := &media.Info{DurationSec: 300, Height: 240, HasAudio: false}
	a := Advise(info, whatsappPreset(), Job{})
	if a.OK {
		t.Fatalf("esperava aviso, got %+v", a)
	}
	if a.Kbps < 260 || a.Kbps > 270 {
		t.Errorf("kbps = %d, expected ~266 without audio", a.Kbps)
	}
	if a.Suggestion.SizeMB < 15 || a.Suggestion.SizeMB > 17 {
		t.Errorf("SizeMB sugerido = %v, esperava ~16", a.Suggestion.SizeMB)
	}
}

func TestAdviseBudgetImpossibleMessage(t *testing.T) {
	// 10 MB for 6h: budget overflows → warning with kbps 0
	info := &media.Info{DurationSec: 21600, Height: 1080, HasAudio: true}
	a := Advise(info, whatsappPreset(), Job{})
	if a.OK || a.Kbps != 0 {
		t.Fatalf("esperava aviso kbps=0, got %+v", a)
	}
	if !strings.Contains(a.Message, "below the ~2000 kbps") {
		t.Errorf("mensagem: %q", a.Message)
	}
	if a.Suggestion.MinutesEach == 0 && a.Suggestion.SizeMB == 0 {
		t.Errorf("expected some suggestion, got %+v", a.Suggestion)
	}
}
