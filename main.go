package main

import (
	"embed"

	"github.com/fabianoflorentino/vidctl/internal/dlog"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// fileDropOption enables the native Wails file drop so absolute paths reach
// the frontend via "wails:file-drop". The DOM-only fallback (text/uri-list) is
// not enough: on WebKitGTK the list comes empty and File objects expose no
// path. Navigation to the dropped file is prevented in the frontend by
// calling preventDefault on dragenter/dragover/drop.
func fileDropOption() *options.DragAndDrop {
	return &options.DragAndDrop{EnableFileDrop: true}
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
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
