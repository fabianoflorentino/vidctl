//go:build !windows

package cmdutil

import (
	"context"
	"os/exec"
)

// Command wraps exec.Command.
func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// CommandContext wraps exec.CommandContext.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// LookPath wraps exec.LookPath.
func LookPath(file string) (string, error) {
	return exec.LookPath(file)
}