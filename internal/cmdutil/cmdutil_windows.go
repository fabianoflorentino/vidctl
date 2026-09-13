//go:build windows

package cmdutil

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const createNoWindow = 0x08000000

var pathMu sync.Mutex

// Command wraps exec.Command hiding any console window on Windows.
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	return cmd
}

// CommandContext wraps exec.CommandContext hiding any console window on Windows.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	return cmd
}

// LookPath resolves a binary against the freshly merged system PATH, so ffmpeg
// installed while the app runs (registry PATH change) is found without restart.
func LookPath(file string) (string, error) {
	pathMu.Lock()
	defer pathMu.Unlock()
	merged := MergePath(
		os.Getenv("PATH"),
		envPath(registry.CURRENT_USER, `Environment`),
		envPath(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`),
		string(os.PathListSeparator),
	)
	os.Setenv("PATH", merged)
	return exec.LookPath(file)
}

func envPath(root registry.Key, key string) string {
	k, err := registry.OpenKey(root, key, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	for _, name := range []string{"Path", "PATH"} {
		v, _, err := k.GetStringValue(name)
		if err == nil {
			return os.ExpandEnv(v)
		}
	}
	return ""
}