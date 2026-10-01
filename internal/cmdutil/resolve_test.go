package cmdutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWithoutOverrideUsesPath(t *testing.T) {
	ClearOverrides()
	dir := t.TempDir()
	fakeBin(t, dir, "ffmpeg", "#!/bin/sh\n")
	got, err := Resolve("ffmpeg")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != filepath.Join(dir, "ffmpeg") {
		t.Errorf("Resolve() = %q, want o binário do PATH %q", got, filepath.Join(dir, "ffmpeg"))
	}
}

func TestResolveMissingWithoutOverride(t *testing.T) {
	ClearOverrides()
	t.Setenv("PATH", t.TempDir())
	if _, err := Resolve("ffmpeg"); err == nil {
		t.Error("Resolve deveria falhar sem override e sem PATH")
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
		t.Error("Override deveria ter sido removido")
	}

	dir := t.TempDir()
	fakeBin(t, dir, "ffmpeg", "#!/bin/sh\n")
	got, err := Resolve("ffmpeg")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != filepath.Join(dir, "ffmpeg") {
		t.Errorf("Resolve() = %q, want o binário do PATH", got)
	}
}

func TestResolveWithMissingOverrideIsAnError(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	// O PATH tem um ffmpeg válido, mas o override aponta para arquivo nenhum:
	// o app precisa avisar em vez de usar silenciosamente o outro binário.
	dir := t.TempDir()
	fakeBin(t, dir, "ffmpeg", "#!/bin/sh\n")

	missing := filepath.Join(t.TempDir(), "nao-existe")
	SetOverride("ffmpeg", missing)

	_, err := Resolve("ffmpeg")
	if err == nil {
		t.Fatal("Resolve deveria falhar com override inválido")
	}
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("erro %v não é ErrNotConfigured", err)
	}
	if !strings.Contains(err.Error(), "ffmpeg") || !strings.Contains(err.Error(), missing) {
		t.Errorf("mensagem %q deveria citar o binário e o caminho", err)
	}
}

func TestResolveWithDirectoryOverrideIsAnError(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	dir := t.TempDir()
	SetOverride("ffmpeg", dir)

	_, err := Resolve("ffmpeg")
	if err == nil {
		t.Fatal("Resolve deveria falhar com override apontando para diretório")
	}
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("erro %v não é ErrNotConfigured", err)
	}
	if !strings.Contains(err.Error(), "diretório") {
		t.Errorf("mensagem %q deveria dizer que é um diretório", err)
	}
}

func TestResolvePathFailureIsNotNotConfigured(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	t.Setenv("PATH", t.TempDir())
	_, err := Resolve("ffprobe")
	if err == nil {
		t.Fatal("Resolve deveria falhar")
	}
	if errors.Is(err, ErrNotConfigured) {
		t.Errorf("falta no PATH não deve virar ErrNotConfigured: %v", err)
	}
}

func TestOverrideReportsConfiguredPath(t *testing.T) {
	ClearOverrides()
	t.Cleanup(ClearOverrides)

	if _, ok := Override("ffprobe"); ok {
		t.Error("ffprobe não deveria ter override no início do teste")
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
		t.Error("ffmpeg ainda tem override")
	}
	if _, ok := Override("ffprobe"); ok {
		t.Error("ffprobe ainda tem override")
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
		t.Error("setar ffmpeg não pode afetar ffprobe")
	}
}

func fakeBin(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}
