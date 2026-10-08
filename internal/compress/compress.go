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
}

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

func bitrateToBits(br string) float64 {
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

// sizeLines represents the bitrate budget computation for size-limited encoding.
type sizeLines struct {
	videoBitrate int
	maxRate      int
	bufSize      int
}

// computeSizeBudget derives video bitrate from the target size and duration.
func computeSizeBudget(job Job, preset presets.Preset, info *media.Info, durationSec float64) (*sizeLines, error) {
	targetBits := preset.SizeMB * 8 * 1024 * 1024
	audioBitsPerSec := bitrateToBits(preset.AudioBitrate)
	if !info.HasAudio {
		audioBitsPerSec = 0
	}
	if durationSec <= 0 {
		return nil, errors.New("duração do vídeo inválida")
	}

	// 5% headroom for container/muxing overhead.
	videoBits := math.Floor(targetBits*0.95 - audioBitsPerSec*durationSec)
	if videoBits <= 0 {
		return nil, fmt.Errorf(
			"tamanho alvo (%.0f MB) pequeno demais para %.0fs de vídeo. Aumente o alvo.",
			preset.SizeMB, durationSec)
	}

	vbr := int(math.Floor(videoBits / durationSec))
	if vbr < 50_000 {
		vbr = 50_000
	}

	return &sizeLines{
		videoBitrate: vbr,
		maxRate:      int(float64(vbr) * 1.5),
		bufSize:      vbr * 2,
	}, nil
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

func seekArgs(job Job, seg split.Segment) []string {
	if job.Split == nil {
		return nil
	}
	return []string{
		"-ss", strconv.FormatFloat(seg.StartSec, 'f', 3, 64),
		"-to", strconv.FormatFloat(seg.EndSec, 'f', 3, 64),
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

	lines, err := computeSizeBudget(job, preset, info, seg.Duration())
	if err != nil {
		return nil, nil, err
	}

	vbr := strconv.Itoa(lines.videoBitrate)
	passLog := passLogFile(jobID)
	seek := seekArgs(job, seg)

	filter := "scale='min(1280,iw)':'min(1280,ih)':force_original_aspect_ratio=decrease"

	pass1Args := []string{
		"-y",
	}
	pass1Args = append(pass1Args, seek...)
	pass1Args = append(pass1Args,
		"-i", job.InputPath,
		"-an",
		"-vf", filter,
		"-c:v", "libx264", "-b:v", vbr,
		"-preset", "medium",
		"-passlogfile", passLog,
		"-pass", "1",
		"-progress", "pipe:1", "-nostats",
		"-f", "null", os.DevNull,
	)

	pass2Args := []string{
		"-y",
	}
	pass2Args = append(pass2Args, seek...)
	pass2Args = append(pass2Args,
		"-i", job.InputPath,
		"-vf", filter,
		"-c:v", "libx264", "-b:v", vbr,
		"-maxrate", strconv.Itoa(lines.maxRate),
		"-bufsize", strconv.Itoa(lines.bufSize),
		"-preset", "medium",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", preset.AudioBitrate,
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

// buildCrfPass creates the single-pass ffmpeg command for quality-based encoding.
func buildCrfPass(ctx context.Context, job Job, preset presets.Preset, info *media.Info, seg split.Segment) *exec.Cmd {
	bin, _ := ffmpegPath()
	args := []string{"-y"}
	args = append(args, seekArgs(job, seg)...)
	args = append(args,
		"-i", job.InputPath,
		"-vf", "scale='min(1280,iw)':'min(1280,ih)':force_original_aspect_ratio=decrease",
		"-c:v", "libx264",
		"-crf", fmt.Sprintf("%d", int(preset.CRF)),
		"-preset", "medium",
		"-pix_fmt", "yuv420p",
	)
	if info.HasAudio {
		args = append(args, "-c:a", "aac", "-b:a", preset.AudioBitrate)
	}
	args = append(args, "-movflags", "+faststart",
		"-progress", "pipe:1", "-nostats",
		job.OutputPath)
	return cmdutil.CommandContext(ctx, bin, args...)
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

	info, err := media.Probe(job.InputPath)
	if err != nil {
		return fail(err)
	}
	preset, ok := EffectivePreset(job)
	if !ok {
		return fail(errors.New("preset desconhecido: " + job.PresetID))
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
		if err := execute(ctx, pass1, jobID, stage1, seg.Duration(), base1, span1); err != nil {
			return wrapStage("pass 1 falhou", suffix, err)
		}

		stage2 := "pass2/2" + suffix
		events.EmitProgress(jobID, events.KindCompress, stage2, base2)
		if err := execute(ctx, pass2, jobID, stage2, seg.Duration(), base2, span2); err != nil {
			return wrapStage("pass 2 falhou", suffix, err)
		}
		removePassLogs(logID)
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
	if err := execute(ctx, cmd, jobID, stage, seg.Duration(), base, span); err != nil {
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
	return preset, true
}
