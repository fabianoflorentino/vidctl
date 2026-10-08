package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/compress"
	"github.com/fabianoflorentino/vidctl/internal/config"
	"github.com/fabianoflorentino/vidctl/internal/estimate"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/sysinfo"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the root binding struct exposed to the frontend.
type App struct {
	ctx   context.Context
	jobs  *compress.Manager
	sys   *sysinfo.Collector
	store *config.Store
}

const videoFilterPattern = "*.mp4;*.mkv;*.mov;*.avi;*.webm;*.m4v;*.ts;*.flv"

// NewApp creates the application struct with the settings file in the default
// user configuration directory. A machine without a writable configuration
// directory still runs: it just does not remember anything between sessions.
func NewApp() *App {
	store, err := config.NewStore("")
	if err != nil {
		store = nil
	}
	return newApp(store)
}

// newApp creates the application with an explicit settings store. Tests pass a
// store rooted in a temporary directory; a nil store disables persistence.
func newApp(store *config.Store) *App {
	return &App{
		jobs:  compress.NewManager(),
		sys:   sysinfo.NewCollector(),
		store: store,
	}
}

// GetUsage returns live system resource usage for the progress panel.
func (a *App) GetUsage() sysinfo.Snapshot {
	return a.sys.Snapshot()
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	events.SetEmitter(func(name string, data any) {
		a.jobs.Observe(name, data)
		wailsruntime.EventsEmit(ctx, name, data)
	})
	cfg := a.loadConfig()
	a.applyToolPaths(cfg)
	a.jobs.SetMaxParallel(cfg.MaxParallel)
}

// loadConfig returns the persisted settings, or the defaults when there is no
// store or the file cannot be read. Loading never fails, so the app always
// starts with a usable configuration.
func (a *App) loadConfig() config.Config {
	if a.store == nil {
		return config.Defaults()
	}
	cfg, _ := a.store.Load()
	return cfg
}

// applyToolPaths points the binary lookup at the configured ffmpeg/ffprobe.
// An unusable path stays configured so the preferences screen can show the
// problem instead of the app silently using a different binary.
func (a *App) applyToolPaths(cfg config.Config) {
	cmdutil.SetOverride("ffmpeg", cfg.FFmpegPath)
	cmdutil.SetOverride("ffprobe", cfg.FFprobePath)
}

// GetConfig returns the persisted settings.
func (a *App) GetConfig() config.Config {
	return a.loadConfig()
}

// SaveConfig persists the settings and applies the tool paths immediately, so
// the next job uses them without a restart.
func (a *App) SaveConfig(cfg config.Config) error {
	if a.store == nil {
		return errors.New("sem pasta de configuração disponível para salvar as preferências")
	}
	if err := a.store.Save(cfg); err != nil {
		return err
	}
	a.applyToolPaths(cfg)
	a.jobs.SetMaxParallel(cfg.MaxParallel)
	return nil
}

// SystemStatus reports the state of required external tools.
type SystemStatus struct {
	FFmpegOK bool   `json:"ffmpegOK"`
	Message  string `json:"message"`
}

// GetAdvice estimates the quality of the current settings before encoding,
// suggesting better parameters when the video bitrate would be too low.
func (a *App) GetAdvice(job compress.Job) (compress.Advice, error) {
	if job.InputPath == "" {
		return compress.Advice{}, errors.New("escolha um vídeo primeiro")
	}
	preset, ok := compress.EffectivePreset(job)
	if !ok {
		return compress.Advice{}, fmt.Errorf("preset desconhecido: %s", job.PresetID)
	}
	info, err := media.Probe(job.InputPath)
	if err != nil {
		return compress.Advice{}, err
	}
	return compress.Advise(info, preset, job), nil
}

// EstimateSize forecasts the output size of the current settings before
// encoding, using the exact budget the 2-pass encoder will spend. CRF mode
// has no deterministic forecast yet.
func (a *App) EstimateSize(job compress.Job) (estimate.Result, error) {
	if job.InputPath == "" {
		return estimate.Result{}, errors.New("escolha um vídeo primeiro")
	}
	preset, ok := compress.EffectivePreset(job)
	if !ok {
		return estimate.Result{}, fmt.Errorf("preset desconhecido: %s", job.PresetID)
	}
	info, err := media.Probe(job.InputPath)
	if err != nil {
		return estimate.Result{}, err
	}
	return estimate.Size(info, preset, job.Split)
}

// CheckFFmpeg verifies that ffmpeg and ffprobe are available, honouring the
// paths configured in the preferences.
func (a *App) CheckFFmpeg() SystemStatus {
	var missing []string
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := cmdutil.Resolve(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		return SystemStatus{
			FFmpegOK: false,
			Message:  "Faltam os programas: " + joinNames(missing) + ". Instale o ffmpeg para usar o vidctl.",
		}
	}
	return SystemStatus{FFmpegOK: true}
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

// GetPresets returns the platform presets available for compression.
func (a *App) GetPresets() []presets.Preset {
	return presets.List()
}

// OpenInputDialog opens the native dialog to pick a source video.
// Returns "" when the user cancels.
func (a *App) OpenInputDialog() (string, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Selecione o vídeo",
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "Vídeos",
			Pattern:     videoFilterPattern,
		}},
	})
	if err != nil {
		return "", fmt.Errorf("falha ao abrir o seletor de arquivos: %w", err)
	}
	return path, nil
}

// OpenMultipleDialog opens the native dialog to pick several source videos.
// Returns an empty slice when the user cancels.
func (a *App) OpenMultipleDialog() ([]string, error) {
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Selecione os vídeos",
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "Vídeos",
			Pattern:     videoFilterPattern,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir o seletor de arquivos: %w", err)
	}
	return paths, nil
}

// OpenOutputDialog opens the native save dialog for the compressed file.
// Returns "" when the user cancels.
func (a *App) OpenOutputDialog(suggestedName string) (string, error) {
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Salvar vídeo comprimido",
		DefaultFilename: suggestedName,
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "Vídeo MP4",
			Pattern:     "*.mp4",
		}},
	})
	if err != nil {
		return "", fmt.Errorf("falha ao abrir o seletor de arquivos: %w", err)
	}
	return path, nil
}

// GetMediaInfo probes a media file and returns its metadata.
func (a *App) GetMediaInfo(path string) (media.Info, error) {
	info, err := media.Probe(path)
	if err != nil {
		return media.Info{}, err
	}
	return *info, nil
}

// GetThumbnail returns a JPEG data-URL with a frame of the video, cached by
// path, size and mtime in the system temp dir.
func (a *App) GetThumbnail(path string, durationSec float64) (string, error) {
	return media.ThumbDataURL(path, durationSec)
}

// Compress enqueues a compression job and returns its ID. The queue emits
// compress:queued with the initial position and runs it when a slot frees up.
func (a *App) Compress(req compress.Job) (string, error) {
	if req.InputPath == "" {
		return "", errors.New("Nenhum vídeo selecionado")
	}
	if req.PresetID == "" {
		return "", errors.New("Nenhum preset selecionado")
	}
	jobID, _ := a.jobs.Enqueue(newTask(req))
	return jobID, nil
}

// JobAck reports the outcome of enqueueing one job of a batch.
type JobAck struct {
	JobID    string `json:"jobId"`
	Position int    `json:"position"`
	Error    string `json:"error"`
}

// CompressMultiple enqueues every job of a batch in order, validating them
// individually so one bad entry does not drop the rest of the batch.
func (a *App) CompressMultiple(reqs []compress.Job) []JobAck {
	acks := make([]JobAck, 0, len(reqs))
	for _, req := range reqs {
		if req.InputPath == "" {
			acks = append(acks, JobAck{Error: "Nenhum vídeo selecionado"})
			continue
		}
		if req.PresetID == "" {
			acks = append(acks, JobAck{Error: "Nenhum preset selecionado"})
			continue
		}
		jobID, position := a.jobs.Enqueue(newTask(req))
		acks = append(acks, JobAck{JobID: jobID, Position: position})
	}
	return acks
}

func newTask(job compress.Job) compress.Task {
	return compress.Task{
		Kind:  events.KindCompress,
		Label: filepath.Base(job.InputPath),
		Job:   job,
	}
}

// GetTasks returns the status snapshot of every task in the queue.
func (a *App) GetTasks() []compress.TaskStatus {
	return a.jobs.List()
}

// ClearFinished drops finished, failed and canceled tasks from the queue,
// keeping the pending ones, and returns what is left.
func (a *App) ClearFinished() []compress.TaskStatus {
	a.jobs.ClearFinished()
	return a.jobs.List()
}

// Cancel aborts a running compression job.
func (a *App) Cancel(jobID string) error {
	a.jobs.Cancel(jobID)
	return nil
}

// OpenFolder reveals the given file's folder in the file manager.
func (a *App) OpenFolder(path string) error {
	dir := filepath.Dir(path)
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}
