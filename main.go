package main

import (
	"embed"
	"runtime"

	"github.com/fabianoflorentino/vidctl/internal/dlog"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// fileDropOption decides how file drops reach the app. On Linux the Wails drop
// handler returns FALSE from the GTK drop signal, which makes WebKit navigate
// to the dropped file (it "plays" the video and the UI is lost), so the Linux
// frontend reads text/uri-list from the DOM drop event instead. Windows and
// macOS keep the native Wails path, which returns absolute paths reliably.
func fileDropOption() *options.DragAndDrop {
	dnd := &options.DragAndDrop{}
	if runtime.GOOS != "linux" {
		dnd.EnableFileDrop = true
	}
	return dnd
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	dlog.Printf("[main] vidctl iniciado com VIDCTL_DEBUG ativo")

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "vidctl — compressor de vídeo",
		Width:     1000,
		Height:    700,
		MinWidth:  720,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 13, G: 12, B: 10, A: 1},
		DragAndDrop:      fileDropOption(),
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
