// Package compress implements the ffmpeg compression pipeline: bitrate
// budgeting, two-pass size-limited encoding and CRF quality encoding.
package compress

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/encode"
	"github.com/fabianoflorentino/vidctl/internal/estimate"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/split"
)

// Job is the payload coming from the frontend.
type Job struct {
	InputPath  string      `json:"inputPath"`
	OutputPath string      `json:"outputPath"`
	PresetID   string      `json:"presetId"`
	SizeMB     float64     `json:"sizeMB"` // used when preset mode is "size"
	CRF        float64     `json:"crf"`
	Split      *split.Spec `json:"split,omitempty"`
	Codec      string      `json:"codec,omitempty"`    // overrides the preset codec when set
	Hardware   string      `json:"hardware,omitempty"` // overrides the preset hardware when set

	// Per-file adjustments (fase 5). All optional.
	Scale         string  `json:"scale,omitempty"`         // "" default cap | ScaleOriginal | "WxH"
	TrimStartSec  float64 `json:"trimStartSec,omitempty"`  // single-window cut, keeps [start, end)
	TrimEndSec    float64 `json:"trimEndSec,omitempty"`    // 0 means "until the end"
	RemoveAudio   bool    `json:"removeAudio,omitempty"`   // drop the audio track
	FPS           float64 `json:"fps,omitempty"`           // 0 keeps the source frame rate
	Rotate        int     `json:"rotate,omitempty"`        // 0|90|180|270 degrees
	ThumbnailPath string  `json:"thumbnailPath,omitempty"` // generate one frame after encoding
}

// ScaleOriginal keeps the source resolution (no scale filter).
const ScaleOriginal = "original"

func ffmpegPath() (string, error) {
	p, err := cmdutil.Resolve("ffmpeg")
	if errors.Is(err, cmdutil.ErrNotConfigured) {
		return "", err
	}
	if err != nil {
		return "", errors.New("ffmpeg não encontrado. Instale o ffmpeg para usar o vidctl.")
	}
	return p, nil
}

func passLogFile(jobID string) string {
	return filepath.Join(os.TempDir(), "vidctl-"+jobID)
}

func removePassLogs(jobID string) {
	base := passLogFile(jobID)
	for _, f := range []string{base, base + "-0.log", base + "-0.log.mbtree"} {
		_ = os.Remove(f)
	}
}

func passLogID(jobID string, job Job, seg split.Segment) string {
	if job.Split == nil {
		return jobID
	}
	return fmt.Sprintf("%s-p%d", jobID, seg.Index)
}

// buildSizePasses creates the two ffmpeg commands for 2-pass size-limited encoding.
func buildSizePasses(ctx context.Context, jobID string, job Job, preset presets.Preset, info *media.Info, seg split.Segment) (*exec.Cmd, *exec.Cmd, error) {
	bin, err := ffmpegPath()
	if err != nil {
		return nil, nil, err
	}

	lines, err := estimate.BudgetFor(preset, info, effectiveDuration(job, seg))
	if err != nil {
		return nil, nil, err
	}

	passLog := passLogFile(jobID)
	seek := seekArgs(job, seg)
	h265 := h265Flag(preset.Codec)

	pass1Args := []string{"-y"}
	pass1Args = append(pass1Args, seek...)
	pass1Args = append(pass1Args, "-i", job.InputPath, "-an")
	pass1Args = appendVideoFilter(pass1Args, job)
	pass1Args = append(pass1Args, encode.Pass1Args(preset.Codec, preset.Hardware, lines.VideoBitrate)...)
	if h265 != nil {
		pass1Args = append(pass1Args, h265...)
	}
	pass1Args = append(pass1Args,
		"-passlogfile", passLog,
		"-pass", "1",
		"-progress", "pipe:1", "-nostats",
		"-f", "null", os.DevNull,
	)

	pass2Args := []string{"-y"}
	pass2Args = append(pass2Args, seek...)
	pass2Args = append(pass2Args, "-i", job.InputPath)
	pass2Args = appendVideoFilter(pass2Args, job)
	pass2Args = append(pass2Args, encode.SizeArgs(preset.Codec, preset.Hardware, lines.VideoBitrate, lines.MaxRate, lines.BufSize)...)
	pass2Args = append(pass2Args, "-pix_fmt", "yuv420p")
	if h265 != nil {
		pass2Args = append(pass2Args, h265...)
	}
	if job.RemoveAudio {
		pass2Args = append(pass2Args, "-an")
	} else {
		pass2Args = append(pass2Args, "-c:a", "aac", "-b:a", preset.AudioBitrate)
	}
	pass2Args = append(pass2Args,
		"-movflags", "+faststart",
		"-passlogfile", passLog,
		"-pass", "2",
		"-progress", "pipe:1", "-nostats",
		job.OutputPath,
	)

	return cmdutil.CommandContext(ctx, bin, pass1Args...),
		cmdutil.CommandContext(ctx, bin, pass2Args...),
		nil
}

// buildSizeCmd creates the single-pass ffmpeg command for size-limited encoding
// on hardware backends that do not support two-pass (NVENC, AMF, VideoToolbox).
func buildSizeCmd(ctx context.Context, job Job, preset presets.Preset, info *media.Info, seg split.Segment) (*exec.Cmd, error) {
	if _, err := ffmpegPath(); err != nil {
		return nil, err
	}
	lines, err := estimate.BudgetFor(preset, info, effectiveDuration(job, seg))
	if err != nil {
		return nil, err
	}
	v := encode.SizeArgs(preset.Codec, preset.Hardware, lines.VideoBitrate, lines.MaxRate, lines.BufSize)
	return buildSingle(ctx, job, preset, info, seg, v), nil
}

// buildCrfPass creates the single-pass ffmpeg command for quality-based encoding.
func buildCrfPass(ctx context.Context, job Job, preset presets.Preset, info *media.Info, seg split.Segment) *exec.Cmd {
	v := encode.QualityArgs(preset.Codec, preset.Hardware, preset.CRF)
	return buildSingle(ctx, job, preset, info, seg, v)
}

// buildSingle assembles a single-pass ffmpeg command around the given -c:v
// section. The ffmpeg binary is resolved anew; a missing binary yields a
// command with an empty path so the error surfaces when ffmpegPath reports it.
func buildSingle(ctx context.Context, job Job, preset presets.Preset, info *media.Info, seg split.Segment, videoArgs []string) *exec.Cmd {
	bin, err := ffmpegPath()
	if err != nil {
		bin = ""
	}
	args := []string{"-y"}
	args = append(args, seekArgs(job, seg)...)
	args = append(args, "-i", job.InputPath)
	args = appendVideoFilter(args, job)
	args = append(args, videoArgs...)
	args = append(args,
		"-pix_fmt", "yuv420p",
	)
	if info.HasAudio && !job.RemoveAudio {
		args = append(args, "-c:a", "aac", "-b:a", preset.AudioBitrate)
	}
	if h := h265Flag(preset.Codec); h != nil {
		args = append(args, h...)
	}
	args = append(args, "-movflags", "+faststart",
		"-progress", "pipe:1", "-nostats",
		job.OutputPath)
	return cmdutil.CommandContext(ctx, bin, args...)
}

// buildThumbnail assembles the post-encode command that writes a single frame
// of the finished output to thumbPath.
func buildThumbnail(ctx context.Context, outPath, thumbPath string, posSec float64) (*exec.Cmd, error) {
	bin, err := ffmpegPath()
	if err != nil {
		return nil, err
	}
	args := []string{
		"-y",
		"-ss", formatSec(posSec),
		"-i", outPath,
		"-frames:v", "1",
		thumbPath,
	}
	return cmdutil.CommandContext(ctx, bin, args...), nil
}

// runQuiet executes a command without progress scanning.
func runQuiet(ctx context.Context, cmd *exec.Cmd) error {
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return err
	}
	return nil
}

// h265Flag silences the per-frame stats libx265 writes to stderr.
func h265Flag(codec string) []string {
	if codec != encode.CodecH265 {
		return nil
	}
	return []string{"-x265-params", "log-level=error"}
}

// execute runs a command, scanning `-progress pipe:1` and emitting progress events.
// The emitted percent is remapped to [base, base+span] so multi-segment jobs can
// show aggregated progress.
func execute(ctx context.Context, cmd *exec.Cmd, jobID, stage string, duration, base, span float64) error {
	progressPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(progressPipe)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			_ = cmd.Process.Kill()
			return err
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") {
			outTimeUS, _ := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_us="), 64)
			pct := math.Min(outTimeUS/(duration*1_000_000), 0.999)
			events.EmitProgress(jobID, events.KindCompress, stage, base+pct*span)
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return err
	}
	return nil
}

// Run validates the job and executes the ffmpeg pipeline, emitting events.
// When job.Split is set, the video is cut into sequential parts, each encoded
// separately; a compress:done event is emitted per part. Failures emit
// compress:error and are returned; cancellation is returned silently.
func Run(ctx context.Context, jobID string, job Job) error {
	fail := func(err error) error {
		events.EmitError(jobID, events.KindCompress, err.Error())
		return err
	}
	if job.InputPath == "" || job.OutputPath == "" {
		return fail(errors.New("caminho de entrada ou saída não pode ser vazio"))
	}
	if job.InputPath == job.OutputPath {
		return fail(errors.New("o arquivo de saída não pode ser igual ao de entrada"))
	}
	if err := Validate(job); err != nil {
		return fail(err)
	}

	info, err := media.Probe(job.InputPath)
	if err != nil {
		return fail(err)
	}
	preset, ok := EffectivePreset(job)
	if !ok {
		return fail(errors.New("preset desconhecido: " + job.PresetID))
	}
	if !encode.ValidCodec(preset.Codec) {
		return fail(errors.New("codec desconhecido: " + preset.Codec))
	}
	if !encode.ValidHardware(preset.Hardware) {
		return fail(errors.New("codificador de hardware desconhecido: " + preset.Hardware))
	}

	if preset.Mode == "size" && preset.SizeMB <= 0 {
		return fail(errors.New("tamanho alvo deve ser maior que zero"))
	}

	segs := []split.Segment{{Index: 1, StartSec: 0, EndSec: info.DurationSec}}
	if job.Split != nil {
		planned, err := split.Plan(info.DurationSec, *job.Split)
		if err != nil {
			return fail(err)
		}
		segs = planned
	}

	if err := os.MkdirAll(filepath.Dir(job.OutputPath), 0o755); err != nil {
		return fail(fmt.Errorf("falha ao criar pasta de saída: %w", err))
	}

	total := len(segs)
	segSpan := 100.0 / float64(total)

	for _, seg := range segs {
		outPath := job.OutputPath
		if total > 1 {
			outPath = split.Filename(job.OutputPath, seg.Index, total)
		}
		segJob := job
		segJob.OutputPath = outPath
		logID := passLogID(jobID, job, seg)

		if err := encodeSegment(ctx, jobID, logID, segJob, preset, info, seg, total, segSpan); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if errors.Is(err, context.Canceled) {
				return err
			}
			return fail(err)
		}

		var sizeBytes int64
		var sizeMB float64
		if out, err := os.Stat(outPath); err == nil {
			sizeBytes = out.Size()
			sizeMB = float64(sizeBytes) / (1024 * 1024)
		}
		if job.ThumbnailPath != "" && seg.Index == total {
			thumb, err := buildThumbnail(ctx, outPath, job.ThumbnailPath, effectiveDuration(job, seg)/2)
			if err != nil {
				return fail(err)
			}
			if err := runQuiet(ctx, thumb); err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return fail(fmt.Errorf("falha ao gerar thumbnail: %w", err))
			}
		}
		events.EmitDone(jobID, events.KindCompress, outPath, sizeBytes, sizeMB)
	}
	return nil
}

func encodeSegment(ctx context.Context, jobID, logID string, job Job, preset presets.Preset, info *media.Info, seg split.Segment, total int, segSpan float64) error {
	suffix := ""
	if total > 1 {
		suffix = fmt.Sprintf(" · parte %d/%d", seg.Index, total)
	}

	if preset.Mode == "size" {
		removePassLogs(logID)
		if encode.TwoPass(preset.Hardware) {
			pass1, pass2, err := buildSizePasses(ctx, logID, job, preset, info, seg)
			if err != nil {
				return err
			}

			base1, span1 := 0.0, 100.0
			base2, span2 := 0.0, 100.0
			if job.Split != nil {
				base := float64(seg.Index-1) * segSpan
				base1, span1 = base, segSpan/2
				base2, span2 = base+segSpan/2, segSpan/2
			}

			stage1 := "pass1/2" + suffix
			events.EmitProgress(jobID, events.KindCompress, stage1, base1)
			if err := execute(ctx, pass1, jobID, stage1, effectiveDuration(job, seg), base1, span1); err != nil {
				return wrapStage("pass 1 falhou", suffix, err)
			}

			stage2 := "pass2/2" + suffix
			events.EmitProgress(jobID, events.KindCompress, stage2, base2)
			if err := execute(ctx, pass2, jobID, stage2, effectiveDuration(job, seg), base2, span2); err != nil {
				return wrapStage("pass 2 falhou", suffix, err)
			}
			removePassLogs(logID)
			return nil
		}

		cmd, err := buildSizeCmd(ctx, job, preset, info, seg)
		if err != nil {
			return err
		}
		base, span := 0.0, 100.0
		if job.Split != nil {
			base = float64(seg.Index-1) * segSpan
			span = segSpan
		}
		stage := "encoding" + suffix
		events.EmitProgress(jobID, events.KindCompress, stage, base)
		if err := execute(ctx, cmd, jobID, stage, effectiveDuration(job, seg), base, span); err != nil {
			return wrapStage("falha ao comprimir", suffix, err)
		}
		return nil
	}

	base, span := 0.0, 100.0
	if job.Split != nil {
		base = float64(seg.Index-1) * segSpan
		span = segSpan
	}
	cmd := buildCrfPass(ctx, job, preset, info, seg)
	stage := "encoding" + suffix
	events.EmitProgress(jobID, events.KindCompress, stage, base)
	if err := execute(ctx, cmd, jobID, stage, effectiveDuration(job, seg), base, span); err != nil {
		return wrapStage("falha ao comprimir", suffix, err)
	}
	return nil
}

func wrapStage(msg, suffix string, err error) error {
	return fmt.Errorf("%s%s: %w", msg, suffix, err)
}

// EffectivePreset resolves the preset applying the user's overrides from the job.
func EffectivePreset(job Job) (presets.Preset, bool) {
	preset, ok := presets.ByID(job.PresetID)
	if !ok {
		return presets.Preset{}, false
	}
	if preset.Mode == "crf" && job.CRF > 0 {
		preset.CRF = job.CRF
	}
	if preset.Mode == "size" && job.SizeMB > 0 {
		preset.SizeMB = job.SizeMB
	}
	if job.Codec != "" {
		preset.Codec = job.Codec
	}
	if job.Hardware != "" {
		preset.Hardware = job.Hardware
	}
	return preset, true
}
