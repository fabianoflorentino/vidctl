package cmdutil

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// The overrides let the user point the app at an ffmpeg/ffprobe outside the
// system PATH. They are process-wide on purpose: media.Probe and compress.Run
// are package-level helpers with no place to thread a resolver through, and the
// values only change when the preferences are saved or at startup.

var (
	overrideMu sync.RWMutex
	overrides  = make(map[string]string)
)

// SetOverride forces Resolve to return path for the binary called name. An
// empty path removes a previously set override. The path is not validated here
// so a value that is not usable yet still survives a restart; Resolve reports
// the problem on the next lookup.
func SetOverride(name, path string) {
	overrideMu.Lock()
	defer overrideMu.Unlock()
	if path == "" {
		delete(overrides, name)
		return
	}
	overrides[name] = path
}

// Override returns the path configured for name and whether one is set.
func Override(name string) (string, bool) {
	overrideMu.RLock()
	defer overrideMu.RUnlock()
	path, ok := overrides[name]
	return path, ok
}

// ClearOverrides removes every configured path.
func ClearOverrides() {
	overrideMu.Lock()
	defer overrideMu.Unlock()
	overrides = make(map[string]string)
}

// ErrNotConfigured reports that the configured override cannot be used. Callers
// use it to tell a bad user-provided path apart from a binary missing from PATH,
// which deserves a different hint.
var ErrNotConfigured = errors.New("caminho configurado inválido")

// Resolve locates a binary, preferring the configured override over the system
// PATH. An override pointing at a missing file is an error, not a silent
// fallback: the user asked for that specific binary.
func Resolve(name string) (string, error) {
	path, ok := Override(name)
	if !ok || path == "" {
		return LookPath(name)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%w para %s (%s): %w", ErrNotConfigured, name, path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w para %s (%s): é um diretório", ErrNotConfigured, name, path)
	}
	return path, nil
}
