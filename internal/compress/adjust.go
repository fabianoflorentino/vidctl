package compress

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fabianoflorentino/vidctl/internal/split"
)

// defaultScaleFilter caps the output at 1280px on the longer side, preserving
// the aspect ratio. It is the historical behaviour when no scale is chosen.
const defaultScaleFilter = "scale='min(1280,iw)':'min(1280,ih)':force_original_aspect_ratio=decrease"

// buildVideoFilter composes the -vf chain in a fixed order:
// transpose (rotation) → scale → fps. An empty string means "no filter", in
// which case the caller must omit the -vf flag entirely.
func buildVideoFilter(job Job) string {
	parts := make([]string, 0, 3)
	if f := rotateFilter(job.Rotate); f != "" {
		parts = append(parts, f)
	}
	if f := scaleFilter(job.Scale); f != "" {
		parts = append(parts, f)
	}
	if job.FPS > 0 {
		parts = append(parts, "fps="+formatSec(job.FPS))
	}
	return strings.Join(parts, ",")
}

// rotateFilter maps the requested orientation to ffmpeg transpose filters:
// 90°→transpose=1, 180°→transpose=2, 270°→transpose=3.
func rotateFilter(deg int) string {
	switch deg {
	case 90:
		return "transpose=1"
	case 180:
		return "transpose=2"
	case 270:
		return "transpose=3"
	default:
		return ""
	}
}

// scaleFilter returns the scale expression for the requested value. Blank keeps
// the default 1280px cap, ScaleOriginal keeps the source size and any other
// value is treated as WxH with the aspect ratio preserved.
func scaleFilter(scale string) string {
	switch scale {
	case "":
		return defaultScaleFilter
	case ScaleOriginal:
		return ""
	default:
		return "scale=" + scale + ":force_original_aspect_ratio=decrease"
	}
}

// appendVideoFilter appends -vf <chain> only when a filter is actually needed.
func appendVideoFilter(args []string, job Job) []string {
	if f := buildVideoFilter(job); f != "" {
		args = append(args, "-vf", f)
	}
	return args
}

// seekArgs places the input seek flags (-ss/-to) before -i. Split segments take
// precedence; otherwise the single-window trim is used.
func seekArgs(job Job, seg split.Segment) []string {
	if job.Split != nil {
		return []string{"-ss", formatSec(seg.StartSec), "-to", formatSec(seg.EndSec)}
	}
	args := make([]string, 0, 4)
	if job.TrimStartSec > 0 {
		args = append(args, "-ss", formatSec(job.TrimStartSec))
	}
	if job.TrimEndSec > 0 {
		args = append(args, "-to", formatSec(job.TrimEndSec))
	}
	return args
}

// effectiveDuration returns the length the encoder will actually produce, so
// the size budget and the progress bar reflect a trimmed window.
func effectiveDuration(job Job, seg split.Segment) float64 {
	if job.Split == nil && (job.TrimStartSec > 0 || job.TrimEndSec > 0) {
		end := job.TrimEndSec
		if end <= 0 {
			end = seg.EndSec
		}
		return end - job.TrimStartSec
	}
	return seg.Duration()
}

// Validate rejects inconsistent per-file adjustments before a job is queued.
func Validate(job Job) error {
	if job.TrimStartSec < 0 || job.TrimEndSec < 0 {
		return errors.New("os tempos de corte não podem ser negativos")
	}
	if job.TrimEndSec > 0 && job.TrimEndSec <= job.TrimStartSec {
		return errors.New("o fim do corte deve ser maior que o início")
	}
	switch job.Rotate {
	case 0, 90, 180, 270:
	default:
		return fmt.Errorf("rotação inválida: %d° (use 0, 90, 180 ou 270)", job.Rotate)
	}
	if job.FPS < 0 {
		return errors.New("FPS inválido")
	}
	if job.Scale != "" && job.Scale != ScaleOriginal && !strings.Contains(job.Scale, "x") {
		return fmt.Errorf("escala inválida: %q (use LARGURAxALTURA)", job.Scale)
	}
	if job.Split != nil && (job.TrimStartSec > 0 || job.TrimEndSec > 0) {
		return errors.New("corte manual e divisão em partes não podem ser combinados")
	}
	return nil
}

// formatSec renders a duration in seconds for ffmpeg with millisecond precision.
func formatSec(sec float64) string {
	return strconv.FormatFloat(sec, 'f', 3, 64)
}
