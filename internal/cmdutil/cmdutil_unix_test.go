//go:build !windows

package cmdutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFakeBin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-cmdutil-bin")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nprintf ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLookPath(t *testing.T) {
	dir := writeFakeBin(t)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := LookPath("fake-cmdutil-bin")
	if err != nil {
		t.Fatalf("LookPath() error: %v", err)
	}
	if want := filepath.Join(dir, "fake-cmdutil-bin"); got != want {
		t.Errorf("LookPath() = %q, want %q", got, want)
	}
}

func TestLookPathMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := LookPath("cmdutil-no-such-bin-xyz"); err == nil {
		t.Fatal("LookPath() expected error for missing binary")
	}
}

func TestCommandRuns(t *testing.T) {
	dir := writeFakeBin(t)
	cmd := Command(filepath.Join(dir, "fake-cmdutil-bin"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command() run error: %v", err)
	}
	if strings.TrimSpace(string(out)) != "ok" {
		t.Errorf("Command() output = %q, want %q", out, "ok")
	}
}

func TestCommandContextCanceled(t *testing.T) {
	dir := writeFakeBin(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := CommandContext(ctx, filepath.Join(dir, "fake-cmdutil-bin"))
	if err := cmd.Run(); err == nil {
		t.Fatal("CommandContext() expected error when context is canceled")
	}
}
