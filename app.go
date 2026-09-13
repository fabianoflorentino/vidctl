package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/compress"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
	"github.com/fabianoflorentino/vidctl/internal/presets"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the root binding struct exposed to the frontend.
type App struct {
	ctx  context.Context
	jobs *compress.Manager
}

// NewApp creates the application struct.
func NewApp() *App {
	return &App{jobs: compress.NewManager()}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	events.SetEmitter(func(name string, data any) {
		wailsruntime.EventsEmit(ctx, name, data)
	})
}

// SystemStatus reports the state of required external tools.
type SystemStatus struct {
	FFmpegOK bool   `json:"ffmpegOK"`
	Message  string `json:"message"`
}

// CheckFFmpeg verifies that ffmpeg and ffprobe are available.
func (a *App) CheckFFmpeg() SystemStatus {
	var missing []string
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := cmdutil.LookPath(bin); err != nil {
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
			Pattern:     "*.mp4;*.mkv;*.mov;*.avi;*.webm;*.m4v;*.ts;*.flv",
		}},
	})
	if err != nil {
		return "", fmt.Errorf("falha ao abrir o seletor de arquivos: %w", err)
	}
	return path, nil
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

// Compress starts an async compression job and returns its ID.
func (a *App) Compress(req compress.Job) (string, error) {
	if req.InputPath == "" {
		return "", errors.New("Nenhum vídeo selecionado")
	}
	if req.PresetID == "" {
		return "", errors.New("Nenhum preset selecionado")
	}

	jobID := fmt.Sprintf("%d", time.Now().UnixNano())

	jobCtx, cancel := context.WithCancel(context.Background())
	a.jobs.Register(jobID, cancel)

	go func() {
		defer a.jobs.Remove(jobID)
		compress.Run(jobCtx, jobID, req)
	}()

	return jobID, nil
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
