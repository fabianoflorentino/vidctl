package app

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fabianoflorentino/vidctl/internal/audio"
	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/compress"
	"github.com/fabianoflorentino/vidctl/internal/config"
	"github.com/fabianoflorentino/vidctl/internal/dlog"
	"github.com/fabianoflorentino/vidctl/internal/encode"
	"github.com/fabianoflorentino/vidctl/internal/estimate"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/notify"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	"github.com/fabianoflorentino/vidctl/internal/sysinfo"
	"github.com/fabianoflorentino/vidctl/internal/systemtheme"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App exposes the application operations to the frontend through Wails
// bindings: dialogs, media probing, compression queue and settings.
type App struct {
	ctx   context.Context
	jobs  *compress.Manager
	sys   *sysinfo.Collector
	store *config.Store
	enc   *encode.Detector
}

const videoFilterPattern = "*.mp4;*.mkv;*.mov;*.avi;*.webm;*.m4v;*.ts;*.flv"

// NewApp returns an App whose settings live in the default user configuration
// directory. A machine without a writable configuration directory still runs —
// it just does not remember anything between sessions.
func NewApp() *App {
	store, err := config.NewStore("")
	if err != nil {
		store = nil
	}
	return newApp(store)
}

// newApp returns an App with an explicit settings store; a nil store disables
// persistence. Tests pass a store rooted in a temporary directory.
func newApp(store *config.Store) *App {
	a := &App{
		jobs:  compress.NewManager(),
		sys:   sysinfo.NewCollector(),
		store: store,
		enc:   &encode.Detector{},
	}
	a.jobs.SetOnFinish(a.notifyFinish)
	return a
}

// notifyFinish applies the notification preferences when a task reaches a
// terminal state: a desktop alert on success or failure, and an automatic
// reveal of the output folder when the job succeeded. It is a side effect of
// the queue finishing the task, never part of the job's own result.
func (a *App) notifyFinish(kind, label, state, outputPath string, err error) {
	cfg := a.loadConfig()
	if cfg.NotifyOnDone {
		message := fmt.Sprintf("%s finished", label)
		if state == compress.StateError {
			message = fmt.Sprintf("%s failed: %v", label, err)
		}
		notify.Send("vidctl", message)
	}
	if cfg.OpenFolderOnDone && state == compress.StateDone && outputPath != "" {
		if err := a.OpenFolder(outputPath); err != nil {
			dlog.Printf("[app] could not open output folder %q: %v", outputPath, err)
		}
	}
}

// GetUsage returns live system resource usage for the progress panel.
func (a *App) GetUsage() sysinfo.Snapshot {
	return a.sys.Snapshot()
}

// startup wires the event emitter (queue observation + frontend dispatch) and
// applies the persisted tool paths and parallelism. It runs on the Wails
// OnStartup hook.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	events.SetEmitter(func(name string, data any) {
		a.jobs.Observe(name, data)
		wailsruntime.EventsEmit(ctx, name, data)
	})
	cfg := a.loadConfig()
	a.applyToolPaths(cfg)
	a.jobs.SetMaxParallel(cfg.MaxParallel)
	dlog.Printf("[app] startup done; maxParallel=%d ffmpeg=%q ffprobe=%q", cfg.MaxParallel, cfg.FFmpegPath, cfg.FFprobePath)
}

// OnStartupFor returns the Wails OnStartup callback wired to a. It is a
// package-level function, so the binding generator never exposes it to the
// frontend (only methods of bound structs become bindings).
func OnStartupFor(a *App) func(context.Context) {
	return a.startup
}

// SystemTheme is exposed to the frontend: "dark" | "light".
func (a *App) SystemTheme() string {
	return systemtheme.Detect()
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

// applyToolPaths points the binary lookup at the configured ffmpeg/ffprobe. An
// unusable path stays configured so the preferences screen can show the problem
// instead of the app silently using a different binary.
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
		return errors.New("no configuration folder available to save preferences")
	}
	if err := a.store.Save(cfg); err != nil {
		return err
	}
	a.applyToolPaths(cfg)
	a.jobs.SetMaxParallel(cfg.MaxParallel)
	return nil
}

// SystemStatus reports the state of the required external tools.
type SystemStatus struct {
	FFmpegOK bool   `json:"ffmpegOK"`
	Message  string `json:"message"`
}

// GetAdvice estimates the quality of the current settings before encoding,
// suggesting better parameters when the video bitrate would be too low.
func (a *App) GetAdvice(job compress.Job) (compress.Advice, error) {
	if job.InputPath == "" {
		return compress.Advice{}, errors.New("pick a video first")
	}
	job, err := a.resolveEncoder(job)
	if err != nil {
		return compress.Advice{}, err
	}
	preset, ok := compress.EffectivePreset(job)
	if !ok {
		return compress.Advice{}, fmt.Errorf("unknown preset: %s", job.PresetID)
	}
	info, err := media.Probe(job.InputPath)
	if err != nil {
		return compress.Advice{}, err
	}
	return compress.Advise(info, preset, job), nil
}

// EstimateSize forecasts the output size of the current settings before
// encoding, using the exact budget the 2-pass encoder will spend. CRF mode has
// no deterministic forecast yet.
func (a *App) EstimateSize(job compress.Job) (estimate.Result, error) {
	if job.InputPath == "" {
		return estimate.Result{}, errors.New("pick a video first")
	}
	job, err := a.resolveEncoder(job)
	if err != nil {
		return estimate.Result{}, err
	}
	preset, ok := compress.EffectivePreset(job)
	if !ok {
		return estimate.Result{}, fmt.Errorf("unknown preset: %s", job.PresetID)
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
			Message:  "Missing programs: " + joinNames(missing) + ". Install ffmpeg to use vidctl.",
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

// GetEncoders probes which hardware encoders this machine can actually use and
// returns them for the frontend selectors. The result is cached until
// RefreshEncoders resets it ("check again").
func (a *App) GetEncoders() (encode.Availability, error) {
	return a.enc.Availability()
}

// RefreshEncoders drops the cached detection and re-probes the encoders.
func (a *App) RefreshEncoders() (encode.Availability, error) {
	a.enc.Refresh()
	return a.enc.Availability()
}

// resolveEncoder validates the job's codec/hardware overrides and turns the
// "auto" hardware sentinel into the first detected backend, falling back to
// software when nothing is probed.
func (a *App) resolveEncoder(job compress.Job) (compress.Job, error) {
	if err := compress.Validate(job); err != nil {
		return job, err
	}
	if !encode.ValidCodec(job.Codec) {
		return job, fmt.Errorf("unknown codec: %s", job.Codec)
	}
	if job.Hardware == "" {
		return job, nil
	}
	if !encode.ValidHardware(job.Hardware) {
		return job, fmt.Errorf("unknown hardware encoder: %s", job.Hardware)
	}
	if job.Hardware != encode.Auto {
		return job, nil
	}
	avail, err := a.enc.Availability()
	if err != nil {
		job.Hardware = ""
		return job, nil
	}
	job.Hardware = encode.ResolveAuto(encoderIDs(avail))
	return job, nil
}

func encoderIDs(avail encode.Availability) []string {
	ids := make([]string, 0, len(avail.Hardware))
	for _, info := range avail.Hardware {
		if len(info.Codecs) > 0 {
			ids = append(ids, info.ID)
		}
	}
	return ids
}

// OpenInputDialog opens the native dialog to pick a source video. Returns ""
// when the user cancels.
func (a *App) OpenInputDialog() (string, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select the video",
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "Videos",
			Pattern:     videoFilterPattern,
		}},
	})
	if err != nil {
		return "", fmt.Errorf("could not open the file picker: %w", err)
	}
	return path, nil
}

// OpenMultipleDialog opens the native dialog to pick several source videos.
// Returns an empty slice when the user cancels.
func (a *App) OpenMultipleDialog() ([]string, error) {
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select the videos",
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "Videos",
			Pattern:     videoFilterPattern,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("could not open the file picker: %w", err)
	}
	return paths, nil
}

// OpenOutputDialog opens the native save dialog for the compressed file.
// Returns "" when the user cancels.
func (a *App) OpenOutputDialog(suggestedName string) (string, error) {
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save the compressed video",
		DefaultFilename: suggestedName,
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "Video MP4",
			Pattern:     "*.mp4",
		}},
	})
	if err != nil {
		return "", fmt.Errorf("could not open the file picker: %w", err)
	}
	return path, nil
}

// OpenThumbnailDialog opens the native save dialog for the generated thumbnail.
// Returns "" when the user cancels.
func (a *App) OpenThumbnailDialog(suggestedName string) (string, error) {
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save thumbnail",
		DefaultFilename: suggestedName,
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "Image",
			Pattern:     "*.png;*.jpg;*.jpeg",
		}},
	})
	if err != nil {
		return "", fmt.Errorf("could not open the file picker: %w", err)
	}
	return path, nil
}

// OpenAudioDialog opens the native save dialog for an extracted track, with a
// filter matching the chosen format's extension.
func (a *App) OpenAudioDialog(suggestedName, codec string) (string, error) {
	ext, err := audio.Extension(codec)
	if err != nil {
		return "", err
	}
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save audio",
		DefaultFilename: suggestedName,
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "Audio",
			Pattern:     "*." + ext,
		}},
	})
	if err != nil {
		return "", fmt.Errorf("could not open the file picker: %w", err)
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
		return "", errors.New("no video selected")
	}
	if req.PresetID == "" {
		return "", errors.New("no preset selected")
	}
	req, err := a.resolveEncoder(req)
	if err != nil {
		return "", err
	}
	jobID, _ := a.jobs.Enqueue(newTask(req))
	dlog.Printf("[app] Compress input=%q preset=%q codec=%q hardware=%q job=%s", req.InputPath, req.PresetID, req.Codec, req.Hardware, jobID)
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
	dlog.Printf("[app] CompressMultiple received=%d", len(reqs))
	acks := make([]JobAck, 0, len(reqs))
	for _, req := range reqs {
		if req.InputPath == "" {
			acks = append(acks, JobAck{Error: "no video selected"})
			continue
		}
		if req.PresetID == "" {
			acks = append(acks, JobAck{Error: "no preset selected"})
			continue
		}
		req, err := a.resolveEncoder(req)
		if err != nil {
			acks = append(acks, JobAck{Error: err.Error()})
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

func newAudioTask(spec audio.Spec) compress.Task {
	return compress.Task{
		Kind:      events.KindAudio,
		Label:     filepath.Base(spec.InputPath),
		InputPath: spec.InputPath,
		Spec:      spec,
	}
}

// ExtractAudio validates the request and enqueues an audio-extraction task,
// returning its job ID. Failures inside the worker (missing audio track,
// ffmpeg errors) surface as the item's error event.
func (a *App) ExtractAudio(req audio.Spec) (string, error) {
	if strings.TrimSpace(req.InputPath) == "" {
		return "", errors.New("no video selected")
	}
	if err := req.Validate(); err != nil {
		return "", err
	}
	jobID, _ := a.jobs.Enqueue(newAudioTask(req))
	dlog.Printf("[app] ExtractAudio input=%q codec=%q out=%q job=%s", req.InputPath, req.Codec, req.OutputPath, jobID)
	return jobID, nil
}

// SuggestAudioOutput proposes the output path of an extraction, in the saved
// output directory or next to the input, using the configured suffix.
func (a *App) SuggestAudioOutput(inputPath, codec string) (string, error) {
	if strings.TrimSpace(inputPath) == "" {
		return "", errors.New("no video selected")
	}
	cfg := a.loadConfig()
	name, err := audio.SuggestOutput(inputPath, codec, cfg.AudioSuffix)
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(cfg.OutputDir)
	if dir == "" {
		dir = filepath.Dir(inputPath)
	}
	return filepath.Join(dir, name), nil
}

// GetTasks returns the status snapshot of every task in the queue.
func (a *App) GetTasks() []compress.TaskStatus {
	tasks := a.jobs.List()
	dlog.Printf("[app] GetTasks -> %d tasks", len(tasks))
	return tasks
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

// DebugEnabled tells the frontend whether local tracing (VIDCTL_DEBUG) is on.
func (a *App) DebugEnabled() bool {
	return dlog.Enabled()
}

// DebugLog records one frontend trace line in the local debug log.
func (a *App) DebugLog(msg string) {
	dlog.Printf("[frontend] %s", msg)
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
