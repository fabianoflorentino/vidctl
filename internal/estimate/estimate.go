// Package estimate forecasts the size of a compression before the encode
// runs. The bitrate budget lives here so the 2-pass encoder and the UI read
// the very same numbers.
package estimate

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/split"
)

// BitrateToBits converts a bitrate string such as "96k" or "1.5m" into bits
// per second. Unparsable values count as zero.
func BitrateToBits(br string) float64 {
	br = strings.TrimSpace(strings.ToLower(br))
	if strings.HasSuffix(br, "k") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(br, "k"), 64)
		return v * 1000
	}
	if strings.HasSuffix(br, "m") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(br, "m"), 64)
		return v * 1_000_000
	}
	v, _ := strconv.ParseFloat(br, 64)
	return v
}

// Budget is the video bitrate budget of a size-limited 2-pass encode.
type Budget struct {
	VideoBitrate int
	MaxRate      int
	BufSize      int
}

// BudgetFor derives the video bitrate budget from the target size and the
// duration of the segment being encoded: 5% of the target stays free for
// container/muxing overhead and the video bitrate never drops below 50 kbps.
func BudgetFor(preset presets.Preset, info *media.Info, durationSec float64) (Budget, error) {
	targetBits := preset.SizeMB * 8 * 1024 * 1024
	audioBitsPerSec := audioBitsPerSec(preset, info)
	if durationSec <= 0 {
		return Budget{}, errors.New("duração do vídeo inválida")
	}

	videoBits := math.Floor(targetBits*0.95 - audioBitsPerSec*durationSec)
	if videoBits <= 0 {
		return Budget{}, fmt.Errorf(
			"tamanho alvo (%.0f MB) pequeno demais para %.0fs de vídeo. Aumente o alvo.",
			preset.SizeMB, durationSec)
	}

	vbr := int(math.Floor(videoBits / durationSec))
	if vbr < 50_000 {
		vbr = 50_000
	}

	return Budget{
		VideoBitrate: vbr,
		MaxRate:      int(float64(vbr) * 1.5),
		BufSize:      vbr * 2,
	}, nil
}

// Result is the size forecast for the current settings.
type Result struct {
	Mode          string  `json:"mode"`
	Available     bool    `json:"available"`
	TargetSizeMB  float64 `json:"targetSizeMB"`
	CurrentSizeMB float64 `json:"currentSizeMB"`
	SavedPct      float64 `json:"savedPct"`
	VideoKbps     int     `json:"videoKbps"`
	AudioKbps     int     `json:"audioKbps"`
}

// Size forecasts the output size before encoding. In size mode the forecast is
// the very budget the 2-pass encoder will spend, headroom included, so it is
// exact for a given target; a split counts one target per part. CRF mode has
// no deterministic estimate (sampled prediction is a stretch goal), so
// Available stays false and only the current size is reported.
func Size(info *media.Info, preset presets.Preset, spec *split.Spec) (Result, error) {
	if info == nil {
		return Result{}, errors.New("vídeo não analisado")
	}
	audioBits := audioBitsPerSec(preset, info)
	res := Result{
		Mode:          preset.Mode,
		CurrentSizeMB: info.SizeMB,
		AudioKbps:     int(math.Round(audioBits / 1000)),
	}
	if preset.Mode != "size" {
		return res, nil
	}

	segs, err := segmentsFor(info, spec)
	if err != nil {
		return Result{}, err
	}

	var totalBits float64
	for i, seg := range segs {
		budget, err := BudgetFor(preset, info, seg.Duration())
		if err != nil {
			return Result{}, err
		}
		if i == 0 {
			res.VideoKbps = budget.VideoBitrate / 1000
		}
		totalBits += (float64(budget.VideoBitrate) + audioBits) * seg.Duration()
	}

	res.Available = true
	res.TargetSizeMB = totalBits / 8 / (1024 * 1024)
	if res.CurrentSizeMB > 0 {
		res.SavedPct = 100 * (1 - res.TargetSizeMB/res.CurrentSizeMB)
	}
	return res, nil
}

func audioBitsPerSec(preset presets.Preset, info *media.Info) float64 {
	if !info.HasAudio {
		return 0
	}
	return BitrateToBits(preset.AudioBitrate)
}

func segmentsFor(info *media.Info, spec *split.Spec) ([]split.Segment, error) {
	if spec == nil {
		return []split.Segment{{Index: 1, EndSec: info.DurationSec}}, nil
	}
	return split.Plan(info.DurationSec, *spec)
}
