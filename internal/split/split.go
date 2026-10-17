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
	// MaxParts caps how many parts a single video may be cut into.
	MaxParts = 60
	// MaxMinutesEach caps the minutes-per-part input.
	MaxMinutesEach = 720
)

// Spec describes the cut request: exactly one of Parts or MinutesEach is set.
type Spec struct {
	Parts       int     `json:"parts"`
	MinutesEach float64 `json:"minutesEach"`
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
// With MinutesEach, slices have fixed length and any remainder becomes its own
// final part, even when shorter than a minute (e.g. a 1:49 video with 1 min per
// part yields 1:00 + 0:49).
func Plan(durationSec float64, spec Spec) ([]Segment, error) {
	if spec.Parts > 0 && spec.MinutesEach > 0 {
		return nil, errors.New("provide only one cut option: number of parts or minutes per part")
	}
	if spec.Parts <= 0 && spec.MinutesEach <= 0 {
		return nil, errors.New("cut requires a number of parts or minutes per part")
	}
	if durationSec <= 0 {
		return nil, errors.New("invalid video duration")
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
		return nil, fmt.Errorf("maximum of %d parts per video", MaxParts)
	}
	slice := durationSec / float64(parts)
	return segments(durationSec, func(i int) float64 {
		return float64(i) * slice
	}, parts), nil
}

func planByMinutes(durationSec float64, minutes float64) ([]Segment, error) {
	if minutes < 1 {
		return nil, errors.New("cada parte precisa ter pelo menos 1 minuto")
	}
	if minutes > MaxMinutesEach {
		return nil, fmt.Errorf("maximum of %d minutes per part", MaxMinutesEach)
	}
	slice := minutes * 60
	full := int(math.Floor(durationSec / slice))
	tail := durationSec - float64(full)*slice

	// Any remainder becomes its own final part, however short.
	total := full
	if tail > 0 {
		total++
	}
	if total < 2 {
		return nil, fmt.Errorf(
			"with %g min per part the whole video fits in 1 part; no cut", minutes)
	}
	if total > MaxParts {
		return nil, fmt.Errorf("cut would create %d parts; the maximum is %d", total, MaxParts)
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
