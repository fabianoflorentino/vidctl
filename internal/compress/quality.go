package compress

import (
	"fmt"
	"math"

	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/split"
)

// maxTargetMB mirrors the frontend slider ceiling for size targets.
const maxTargetMB = 100.0

// Suggestion carries settings the frontend can apply to fix low quality.
// Zero fields mean "no change" for that control.
type Suggestion struct {
	SizeMB      float64 `json:"sizeMB,omitempty"`
	MinutesEach int     `json:"minutesEach,omitempty"`
	Parts       int     `json:"parts,omitempty"`
}

// Advice is the pre-encode quality estimation for the current settings.
type Advice struct {
	OK         bool       `json:"ok"`
	Kbps       int        `json:"kbps"`
	MinKbps    int        `json:"minKbps"`
	Message    string     `json:"message"`
	Suggestion Suggestion `json:"suggestion"`
}

// minKbpsForHeight returns a conservative average video bitrate per
// resolution tier for libx264 at -preset medium.
func minKbpsForHeight(height int) int {
	switch {
	case height >= 1080:
		return 2000
	case height >= 720:
		return 1200
	case height >= 480:
		return 700
	default:
		return 400
	}
}

// Advise estimates the video bitrate the given settings will produce and,
// when it falls under the resolution tier, suggests better parameters.
// The preset must already carry the effective (user-overridden) values.
func Advise(info *media.Info, preset presets.Preset, job Job) Advice {
	if preset.Mode != "size" {
		return Advice{OK: true}
	}

	segSec := info.DurationSec
	if job.Split != nil {
		segs, err := split.Plan(info.DurationSec, *job.Split)
		if err != nil || len(segs) == 0 {
			return Advice{OK: true}
		}
		segSec = segs[0].Duration()
	}

	minKbps := minKbpsForHeight(info.Height)
	audioBits := bitrateToBits(preset.AudioBitrate)
	if !info.HasAudio {
		audioBits = 0
	}

	lines, err := computeSizeBudget(job, preset, info, segSec)
	if err != nil {
		return lowAdvice(0, minKbps, segSec, info, preset, job, audioBits)
	}
	kbps := lines.videoBitrate / 1000
	if kbps >= minKbps {
		return Advice{OK: true, Kbps: kbps, MinKbps: minKbps}
	}
	return lowAdvice(kbps, minKbps, segSec, info, preset, job, audioBits)
}

func lowAdvice(kbps, minKbps int, segSec float64, info *media.Info, preset presets.Preset, job Job, audioBits float64) Advice {
	scope := fmt.Sprintf("para um vídeo de %s", humanSec(info.DurationSec))
	if job.Split != nil {
		scope = fmt.Sprintf("para cada parte de %s", humanSec(segSec))
	}
	msg := fmt.Sprintf("≈%d kbps de vídeo %s — abaixo dos ~%d kbps recomendados para %dp; a qualidade vai cair muito.", kbps, scope, minKbps, info.Height)

	need := &Suggestion{}
	needMB := math.Ceil(((float64(minKbps)*1000 + audioBits) * segSec) / 0.95 / 8 / 1048576)
	switch {
	case needMB <= maxTargetMB:
		need.SizeMB = needMB
		msg += fmt.Sprintf(" Sugestão: suba o alvo para ~%.0f MB.", needMB)
	default:
		perPartSec := (maxTargetMB * 0.95 * 8 * 1048576) / (float64(minKbps)*1000 + audioBits)
		minutes := int(math.Floor(perPartSec / 60))
		segs, err := split.Plan(info.DurationSec, split.Spec{MinutesEach: minutes})
		if minutes >= 1 && err == nil && len(segs) >= 2 && len(segs) <= split.MaxParts {
			need.MinutesEach = minutes
			need.Parts = len(segs)
			msg += fmt.Sprintf(" Sugestão: corte em partes de ~%d min (%d partes).", minutes, len(segs))
		} else {
			msg += " Com o limite de tamanho atual não há como manter a qualidade — prefira um preset CRF (ex.: YouTube)."
		}
	}

	return Advice{OK: false, Kbps: kbps, MinKbps: minKbps, Message: msg, Suggestion: *need}
}

func humanSec(sec float64) string {
	m := int(sec) / 60
	s := int(sec) % 60
	if m == 0 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dmin%02ds", m, s)
}
