package main

import "github.com/fabianoflorentino/vidctl/internal/app"

// App is the Wails binding surface, exposed to the frontend as main.App.
// Embedding the internal facade keeps the generated client path
// (../wailsjs/go/main/App.js) stable while all logic lives in internal/app.
type App struct {
	*app.App
}

func newApp() *App {
	return &App{App: app.NewApp()}
}
