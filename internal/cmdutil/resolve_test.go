package cmdutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveWithoutOverrideUsesPath(t *testing.T) {
	ClearOverrides()
	dir := t.TempDir()
	want := fakeBin(t, dir, "ffmpeg", "#!/bin/sh\n")
	got, err := Resolve("ffmpeg")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != want {
		t.Errorf("Resolve() = %q, want the PATH binary %q", got, want)
	}
}

func TestResolveMissingWithoutOverride(t *testing.T) {
	ClearOverrides()
	t.Setenv("PATH", t.TempDir())
	if _, err := Resolve("ffmpeg"); err == nil {
		t.Error("Resolve should falhar sem override e sem PATH")
	}
}

func TestSetOverrideWinsOverPath(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	other := t.TempDir()
	fakeBin(t, other, "ffmpeg", "#!/bin/sh\n")

	custom := filepath.Join(t.TempDir(), "ffmpeg-custom")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	SetOverride("ffmpeg", custom)

	got, err := Resolve("ffmpeg")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != custom {
		t.Errorf("Resolve() = %q, want o override %q", got, custom)
	}
}

func TestSetOverrideEmptyRemovesPrevious(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	custom := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	SetOverride("ffmpeg", custom)
	SetOverride("ffmpeg", "")

	if _, ok := Override("ffmpeg"); ok {
		t.Error("Override should ter sido removido")
	}

	dir := t.TempDir()
	want := fakeBin(t, dir, "ffmpeg", "#!/bin/sh\n")
	got, err := Resolve("ffmpeg")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != want {
		t.Errorf("Resolve() = %q, want the PATH binary %q", got, want)
	}
}

func TestResolveWithMissingOverrideIsAnError(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	// The PATH has a valid ffmpeg, but the override points at no file:
	// the app must warn instead of silently using the other binary.
	dir := t.TempDir()
	fakeBin(t, dir, "ffmpeg", "#!/bin/sh\n")

	missing := filepath.Join(t.TempDir(), "nao-existe")
	SetOverride("ffmpeg", missing)

	_, err := Resolve("ffmpeg")
	if err == nil {
		t.Fatal("Resolve should fail with an invalid override")
	}
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("error %v is not ErrNotConfigured", err)
	}
	if !strings.Contains(err.Error(), "ffmpeg") || !strings.Contains(err.Error(), missing) {
		t.Errorf("message %q should mention the binary and the path", err)
	}
}

func TestResolveWithDirectoryOverrideIsAnError(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	dir := t.TempDir()
	SetOverride("ffmpeg", dir)

	_, err := Resolve("ffmpeg")
	if err == nil {
		t.Fatal("Resolve should fail with an override pointing at a directory")
	}
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("error %v is not ErrNotConfigured", err)
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("message %q should say it is a directory", err)
	}
}

func TestResolvePathFailureIsNotNotConfigured(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	t.Setenv("PATH", t.TempDir())
	_, err := Resolve("ffprobe")
	if err == nil {
		t.Fatal("Resolve should falhar")
	}
	if errors.Is(err, ErrNotConfigured) {
		t.Errorf("missing from PATH must not become ErrNotConfigured: %v", err)
	}
}

func TestOverrideReportsConfiguredPath(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	if _, ok := Override("ffprobe"); ok {
		t.Error("ffprobe should have no override at the start of the test")
	}
	SetOverride("ffprobe", "/opt/ffprobe")
	path, ok := Override("ffprobe")
	if !ok || path != "/opt/ffprobe" {
		t.Errorf("Override() = %q, %v; want /opt/ffprobe, true", path, ok)
	}
}

func TestClearOverrides(t *testing.T) {
	t.Cleanup(ClearOverrides)
	SetOverride("ffmpeg", "/opt/ffmpeg")
	SetOverride("ffprobe", "/opt/ffprobe")
	ClearOverrides()

	if _, ok := Override("ffmpeg"); ok {
		t.Error("ffmpeg still tem override")
	}
	if _, ok := Override("ffprobe"); ok {
		t.Error("ffprobe still tem override")
	}
}

func TestOverridesAreIndependentPerBinary(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	custom := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	SetOverride("ffmpeg", custom)

	if _, ok := Override("ffprobe"); ok {
		t.Error("setting ffmpeg must not affect ffprobe")
	}
}

// fakeBin writes an executable named after name into dir and puts dir first on
// PATH. On Windows exec.LookPath only finds binaries carrying an extension from
// PATHEXT, so the fake needs the .exe suffix to be found at all.
func fakeBin(t *testing.T, dir, name, content string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return path
}
