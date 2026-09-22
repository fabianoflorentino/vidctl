// Package split plans time-based cutting of a video into sequential parts.
package split

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// MinPartSec is the minimum duration allowed for any part.
	MinPartSec = 60.0
	// MaxParts caps how many parts a single video may be cut into.
	MaxParts = 60
	// MaxMinutesEach caps the minutes-per-part input.
	MaxMinutesEach = 720
)

// Spec describes the cut request: exactly one of Parts or MinutesEach is set.
type Spec struct {
	Parts       int `json:"parts"`
	MinutesEach int `json:"minutesEach"`
}

// Segment is one cut of the video, in seconds, half-open [StartSec, EndSec).
type Segment struct {
	Index    int     `json:"index"`
	StartSec float64 `json:"startSec"`
	EndSec   float64 `json:"endSec"`
}

// Duration returns the segment length in seconds.
func (s Segment) Duration() float64 {
	return s.EndSec - s.StartSec
}

// Plan computes sequential segments covering the whole video.
// With Parts, the video is divided into equal slices of duration/Parts.
// With MinutesEach, slices have fixed length and the remainder becomes its
// own part, unless it would be shorter than MinPartSec — then it is merged
// into the last slice (no part is ever shorter than 1 minute).
func Plan(durationSec float64, spec Spec) ([]Segment, error) {
	if spec.Parts > 0 && spec.MinutesEach > 0 {
		return nil, errors.New("informe apenas uma opção de corte: número de partes ou minutos por parte")
	}
	if spec.Parts <= 0 && spec.MinutesEach <= 0 {
		return nil, errors.New("corte requer número de partes ou minutos por parte")
	}
	if durationSec <= 0 {
		return nil, errors.New("duração do vídeo inválida")
	}
	if durationSec < MinPartSec {
		return nil, fmt.Errorf("vídeo com %.0fs é mais curto que o mínimo de 1 min por parte", durationSec)
	}

	if spec.Parts > 0 {
		return planByParts(durationSec, spec.Parts)
	}
	return planByMinutes(durationSec, spec.MinutesEach)
}

func planByParts(durationSec float64, parts int) ([]Segment, error) {
	if parts < 2 {
		return nil, errors.New("corte em partes precisa de pelo menos 2 partes")
	}
	if parts > MaxParts {
		return nil, fmt.Errorf("máximo de %d partes por vídeo", MaxParts)
	}
	slice := durationSec / float64(parts)
	if slice < MinPartSec {
		return nil, fmt.Errorf(
			"cada parte teria %s; o mínimo é 1 min. Use menos partes",
			humanDuration(slice))
	}
	return segments(durationSec, func(i int) float64 {
		return float64(i) * slice
	}, parts), nil
}

func planByMinutes(durationSec float64, minutes int) ([]Segment, error) {
	if minutes < 1 {
		return nil, errors.New("cada parte precisa ter pelo menos 1 minuto")
	}
	if minutes > MaxMinutesEach {
		return nil, fmt.Errorf("máximo de %d minutos por parte", MaxMinutesEach)
	}
	slice := float64(minutes) * 60
	full := int(math.Floor(durationSec / slice))
	tail := durationSec - float64(full)*slice

	total := full
	switch {
	case tail >= MinPartSec:
		total++ // remainder becomes its own (shorter) part
	case tail > 0:
		// remainder < 1 min: merge into the last slice so no part is too short
	}
	if total < 2 {
		return nil, fmt.Errorf(
			"com %d min por parte o vídeo inteiro cabe em 1 parte; não há corte", minutes)
	}
	if total > MaxParts {
		return nil, fmt.Errorf("corte geraria %d partes; o máximo é %d", total, MaxParts)
	}

	return segments(durationSec, func(i int) float64 {
		return float64(i) * slice
	}, total), nil
}

func segments(durationSec float64, boundary func(i int) float64, count int) []Segment {
	out := make([]Segment, 0, count)
	for i := 0; i < count; i++ {
		start := boundary(i)
		end := boundary(i + 1)
		if i == count-1 || end > durationSec {
			end = durationSec
		}
		out = append(out, Segment{Index: i + 1, StartSec: start, EndSec: end})
	}
	return out
}

// Filename inserts a "-partN" suffix before the extension. When total needs
// more than one digit the index is zero-padded so files sort correctly.
func Filename(path string, index, total int) string {
	ext := filepath.Ext(path)
	width := 1
	if total >= 10 {
		width = 2
	}
	if total >= 100 {
		width = 3
	}
	num := strconv.Itoa(index)
	if len(num) < width {
		num = strings.Repeat("0", width-len(num)) + num
	}
	return strings.TrimSuffix(path, ext) + "-part" + num + ext
}

func humanDuration(sec float64) string {
	if sec >= 60 {
		m := int(sec) / 60
		s := int(sec) % 60
		return fmt.Sprintf("%d min %d s", m, s)
	}
	return fmt.Sprintf("%.0f s", sec)
}
