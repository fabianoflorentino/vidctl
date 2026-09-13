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
	"sync"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
)

// Job is the payload coming from the frontend.
type Job struct {
	InputPath  string  `json:"inputPath"`
	OutputPath string  `json:"outputPath"`
	PresetID   string  `json:"presetId"`
	SizeMB     float64 `json:"sizeMB"` // used when preset mode is "size"
	CRF        float64 `json:"crf"`
}

type runningJob struct {
	id     string
	cancel context.CancelFunc
}

// Manager tracks running compression jobs and provides cancellation.
type Manager struct {
	mu   sync.Mutex
	jobs map[string]*runningJob
}

// NewManager creates a job manager.
func NewManager() *Manager {
	return &Manager{jobs: make(map[string]*runningJob)}
}

// Register stores the cancel function for a job.
func (m *Manager) Register(id string, cancelFn context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[id] = &runningJob{id: id, cancel: cancelFn}
}

// Cancel cancels the given job.
func (m *Manager) Cancel(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[id]; ok {
		j.cancel()
		delete(m.jobs, id)
	}
}

// Remove forgets a finished job.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.jobs, id)
}

func ffmpegPath() (string, error) {
	p, err := cmdutil.LookPath("ffmpeg")
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
func computeSizeBudget(job Job, preset presets.Preset, info *media.Info) (*sizeLines, error) {
	targetBits := preset.SizeMB * 8 * 1024 * 1024
	audioBitsPerSec := bitrateToBits(preset.AudioBitrate)
	if !info.HasAudio {
		audioBitsPerSec = 0
	}
	if info.DurationSec <= 0 {
		return nil, errors.New("duração do vídeo inválida")
	}

	// 5% headroom for container/muxing overhead.
	videoBits := math.Floor(targetBits*0.95 - audioBitsPerSec*info.DurationSec)
	if videoBits <= 0 {
		return nil, fmt.Errorf(
			"tamanho alvo (%.0f MB) pequeno demais para %.0fs de vídeo. Aumente o alvo.",
			preset.SizeMB, info.DurationSec)
	}

	vbr := int(math.Floor(videoBits / info.DurationSec))
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

// buildSizePasses creates the two ffmpeg commands for 2-pass size-limited encoding.
func buildSizePasses(ctx context.Context, jobID string, job Job, preset presets.Preset, info *media.Info) (*exec.Cmd, *exec.Cmd, error) {
	bin, err := ffmpegPath()
	if err != nil {
		return nil, nil, err
	}

	lines, err := computeSizeBudget(job, preset, info)
	if err != nil {
		return nil, nil, err
	}

	vbr := strconv.Itoa(lines.videoBitrate)
	passLog := passLogFile(jobID)

	filter := "scale='min(1280,iw)':'min(1280,ih)':force_original_aspect_ratio=decrease"

	pass1Args := []string{
		"-y", "-i", job.InputPath,
		"-an",
		"-vf", filter,
		"-c:v", "libx264", "-b:v", vbr,
		"-preset", "medium",
		"-passlogfile", passLog,
		"-pass", "1",
		"-progress", "pipe:1", "-nostats",
		"-f", "null", os.DevNull,
	}

	pass2Args := []string{
		"-y", "-i", job.InputPath,
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
	}

	return cmdutil.CommandContext(ctx, bin, pass1Args...),
		cmdutil.CommandContext(ctx, bin, pass2Args...),
		nil
}

// buildCrfPass creates the single-pass ffmpeg command for quality-based encoding.
func buildCrfPass(ctx context.Context, job Job, preset presets.Preset, info *media.Info) *exec.Cmd {
	bin, _ := ffmpegPath()
	args := []string{
		"-y", "-i", job.InputPath,
		"-vf", "scale='min(1280,iw)':'min(1280,ih)':force_original_aspect_ratio=decrease",
		"-c:v", "libx264",
		"-crf", fmt.Sprintf("%d", int(preset.CRF)),
		"-preset", "medium",
		"-pix_fmt", "yuv420p",
	}
	if info.HasAudio {
		args = append(args, "-c:a", "aac", "-b:a", preset.AudioBitrate)
	}
	args = append(args, "-movflags", "+faststart",
		"-progress", "pipe:1", "-nostats",
		job.OutputPath)
	return cmdutil.CommandContext(ctx, bin, args...)
}

// execute runs a command, scanning `-progress pipe:1` and emitting progress events.
func execute(ctx context.Context, cmd *exec.Cmd, jobID, stage string, duration float64) error {
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
			pct := math.Min((outTimeUS/(duration*1_000_000))*100, 99.9)
			events.EmitProgress(jobID, stage, pct)
		}
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// Run validates the job and executes the ffmpeg pipeline, emitting events.
func Run(ctx context.Context, jobID string, job Job) {
	if job.InputPath == "" || job.OutputPath == "" {
		events.EmitError(jobID, "caminho de entrada ou saída não pode ser vazio")
		return
	}
	if job.InputPath == job.OutputPath {
		events.EmitError(jobID, "o arquivo de saída não pode ser igual ao de entrada")
		return
	}

	info, err := media.Probe(job.InputPath)
	if err != nil {
		events.EmitError(jobID, err.Error())
		return
	}

	preset, ok := presets.ByID(job.PresetID)
	if !ok {
		events.EmitError(jobID, "preset desconhecido: "+job.PresetID)
		return
	}

	if preset.Mode == "crf" && job.CRF > 0 {
		preset.CRF = job.CRF
	}
	if preset.Mode == "size" && job.SizeMB > 0 {
		preset.SizeMB = job.SizeMB
	}
	if preset.Mode == "size" && preset.SizeMB <= 0 {
		events.EmitError(jobID, "tamanho alvo deve ser maior que zero")
		return
	}

	if err := os.MkdirAll(filepath.Dir(job.OutputPath), 0o755); err != nil {
		events.EmitError(jobID, "falha ao criar pasta de saída: "+err.Error())
		return
	}

	if preset.Mode == "size" {
		removePassLogs(jobID)
		pass1, pass2, err := buildSizePasses(ctx, jobID, job, preset, info)
		if err != nil {
			events.EmitError(jobID, err.Error())
			return
		}

		events.EmitProgress(jobID, "pass1/2", 0)
		if err := execute(ctx, pass1, jobID, "pass1/2", info.DurationSec); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			events.EmitError(jobID, "pass 1 falhou: "+err.Error())
			return
		}

		events.EmitProgress(jobID, "pass2/2", 0)
		if err := execute(ctx, pass2, jobID, "pass2/2", info.DurationSec); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			events.EmitError(jobID, "pass 2 falhou: "+err.Error())
			return
		}
		removePassLogs(jobID)
	} else {
		cmd := buildCrfPass(ctx, job, preset, info)
		events.EmitProgress(jobID, "encoding", 0)
		if err := execute(ctx, cmd, jobID, "encoding", info.DurationSec); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			events.EmitError(jobID, "falha ao comprimir: "+err.Error())
			return
		}
	}

	var sizeBytes int64
	var sizeMB float64
	if out, err := os.Stat(job.OutputPath); err == nil {
		sizeBytes = out.Size()
		sizeMB = float64(sizeBytes) / (1024 * 1024)
	}
	events.EmitDone(jobID, job.OutputPath, sizeBytes, sizeMB)
}
