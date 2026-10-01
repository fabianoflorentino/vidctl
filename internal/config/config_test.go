package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.PresetID != "whatsapp-status" {
		t.Errorf("PresetID = %q, want whatsapp-status", cfg.PresetID)
	}
	if cfg.SizeMB != 10 {
		t.Errorf("SizeMB = %v, want 10", cfg.SizeMB)
	}
	if cfg.CRF != 23 {
		t.Errorf("CRF = %v, want 23", cfg.CRF)
	}
	if cfg.Language != "pt" {
		t.Errorf("Language = %q, want pt", cfg.Language)
	}
	if cfg.MaxParallel != 1 {
		t.Errorf("MaxParallel = %d, want 1", cfg.MaxParallel)
	}
	if cfg.OutputDir != "" || cfg.FFmpegPath != "" || cfg.FFprobePath != "" {
		t.Errorf("defaults devem deixar os caminhos vazios: %+v", cfg)
	}
	if cfg.NotifyOnDone || cfg.OpenFolderOnDone {
		t.Errorf("as notificações devem começar desligadas: %+v", cfg)
	}
}

func TestDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if filepath.Base(got) != DirName {
		t.Errorf("Dir() = %q, want terminar em %q", got, DirName)
	}
}

func TestDirWithoutHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("APPDATA", "")

	if _, err := Dir(); err == nil {
		t.Error("Dir deveria falhar sem nenhuma pasta de configuração")
	}
	if _, err := NewStore(""); err == nil {
		t.Error("NewStore(\"\") deveria falhar sem nenhuma pasta de configuração")
	}
}

func TestNewStoreUsesDefaultDirWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	store, err := NewStore("")
	if err != nil {
		t.Fatalf("NewStore(\"\"): %v", err)
	}
	want := filepath.Join(dir, DirName, FileName)
	if store.Path() != want {
		t.Errorf("Path() = %q, want %q", store.Path(), want)
	}
}

func TestNewStoreExplicitDir(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if want := filepath.Join(dir, FileName); store.Path() != want {
		t.Errorf("Path() = %q, want %q", store.Path(), want)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	store := newTestStore(t)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() com arquivo ausente: %v", err)
	}
	if cfg != Defaults() {
		t.Errorf("Load() = %+v, want defaults %+v", cfg, Defaults())
	}
}

func TestLoadCorruptFileReturnsDefaults(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"json invalido", "{ isto nao e json"},
		{"json truncado", `{"presetId": "youtube"`},
		{"array no lugar do objeto", `["whatsapp-status"]`},
		{"vazio", ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			if err := os.WriteFile(store.Path(), []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := store.Load()
			if err != nil {
				t.Fatalf("Load() com arquivo corrompido não pode falhar: %v", err)
			}
			if cfg != Defaults() {
				t.Errorf("Load() = %+v, want defaults", cfg)
			}
		})
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	store := newTestStore(t)
	want := Config{
		PresetID:         "youtube",
		SizeMB:           37.5,
		CRF:              19,
		OutputDir:        "/tmp/vidctl-out",
		FFmpegPath:       "/opt/ffmpeg/bin/ffmpeg",
		FFprobePath:      "/opt/ffmpeg/bin/ffprobe",
		Language:         "en",
		MaxParallel:      3,
		NotifyOnDone:     true,
		OpenFolderOnDone: true,
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("roundtrip = %+v, want %+v", got, want)
	}
}

func TestSaveCreatesDirectoryAndPrivateFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", DirName)
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.Save(Defaults()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("arquivo não criado: %v", err)
	}
	if runtime := fileMode(t, store.Path()); runtime != 0o600 {
		t.Errorf("permissões = %o, want 600", runtime)
	}
}

func TestSaveOverwritesPreviousFile(t *testing.T) {
	store := newTestStore(t)
	first := Defaults()
	first.PresetID = "youtube"
	if err := store.Save(first); err != nil {
		t.Fatalf("primeiro Save: %v", err)
	}
	second := Defaults()
	second.PresetID = "shorts"
	if err := store.Save(second); err != nil {
		t.Fatalf("segundo Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.PresetID != "shorts" {
		t.Errorf("PresetID = %q, want shorts", got.PresetID)
	}
}

func TestSaveLeavesNoTempFileBehind(t *testing.T) {
	store := newTestStore(t)
	if err := store.Save(Defaults()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("arquivo temporário sobrou: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("diretório tem %d arquivos, want só o config: %v", len(entries), entries)
	}
}

func TestSaveValidatesBeforeWriting(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantMsg string
	}{
		{"preset vazio", Config{PresetID: "  ", Language: "pt", MaxParallel: 1}, "presetId"},
		{"size negativo", Config{PresetID: "custom", SizeMB: -1, Language: "pt", MaxParallel: 1}, "sizeMB"},
		{"size acima do limite", Config{PresetID: "custom", SizeMB: MaxSizeMB + 1, Language: "pt", MaxParallel: 1}, "sizeMB"},
		{"crf acima do limite", Config{PresetID: "custom", CRF: 52, Language: "pt", MaxParallel: 1}, "crf"},
		{"crf negativo", Config{PresetID: "custom", CRF: -3, Language: "pt", MaxParallel: 1}, "crf"},
		{"paralelismo zero", Config{PresetID: "custom", Language: "pt", MaxParallel: 0}, "maxParallel"},
		{"idioma desconhecido", Config{PresetID: "custom", Language: "fr", MaxParallel: 1}, "language"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			err := store.Save(tt.cfg)
			if err == nil {
				t.Fatal("Save deveria falhar")
			}
			if !errors.Is(err, ErrInvalidConfig) {
				t.Errorf("erro %v não é ErrInvalidConfig", err)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("mensagem %q não menciona %q", err, tt.wantMsg)
			}
			if _, statErr := os.Stat(store.Path()); statErr == nil {
				t.Error("Save inválido não pode criar o arquivo")
			}
		})
	}
}

func TestSaveRejectsUnwritableDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignora as permissões do diretório")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(filepath.Join(dir, DirName))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.Save(Defaults()); err == nil {
		t.Error("Save em diretório sem escrita deveria falhar")
	}
}

func TestSaveFailsWhenTargetIsADirectory(t *testing.T) {
	store := newTestStore(t)
	// O rename não substitui um diretório por um arquivo.
	if err := os.MkdirAll(store.Path(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(Defaults()); err == nil {
		t.Error("Save deveria falhar quando o destino é um diretório")
	}
	entries, err := os.ReadDir(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("arquivo temporário sobrou após falha: %s", e.Name())
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		name string
		in   Config
		want Config
	}{
		{
			// SizeMB e CRF ficam em 0 de propósito: 0 significa "não usado" e
			// compress.EffectivePreset só aplica o override quando > 0.
			name: "campos de texto vazios voltam ao default",
			in:   Config{},
			want: Config{PresetID: "whatsapp-status", Language: "pt", MaxParallel: 1},
		},
		{
			name: "espaços em volta dos caminhos são removidos",
			in:   Config{PresetID: "youtube", OutputDir: "  /tmp  ", FFmpegPath: " /opt/ffmpeg ", Language: "EN", MaxParallel: 2},
			want: Config{PresetID: "youtube", OutputDir: "/tmp", FFmpegPath: "/opt/ffmpeg", Language: "en", MaxParallel: 2},
		},
		{
			name: "crf acima do limite é limitado",
			in:   Config{PresetID: "custom", CRF: 99, Language: "pt", MaxParallel: 1},
			want: Config{PresetID: "custom", CRF: MaxCRF, Language: "pt", MaxParallel: 1},
		},
		{
			name: "crf negativo é limitado",
			in:   Config{PresetID: "custom", CRF: -10, Language: "pt", MaxParallel: 1},
			want: Config{PresetID: "custom", CRF: MinCRF, Language: "pt", MaxParallel: 1},
		},
		{
			name: "size negativo é limitado a zero",
			in:   Config{PresetID: "custom", SizeMB: -5, Language: "pt", MaxParallel: 1},
			want: Config{PresetID: "custom", SizeMB: 0, Language: "pt", MaxParallel: 1},
		},
		{
			name: "size acima do limite é limitado",
			in:   Config{PresetID: "custom", SizeMB: MaxSizeMB * 4, Language: "pt", MaxParallel: 1},
			want: Config{PresetID: "custom", SizeMB: MaxSizeMB, Language: "pt", MaxParallel: 1},
		},
		{
			name: "paralelismo abaixo de 1 vira 1",
			in:   Config{PresetID: "custom", Language: "pt", MaxParallel: -2},
			want: Config{PresetID: "custom", Language: "pt", MaxParallel: 1},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Normalize(); got != tt.want {
				t.Errorf("Normalize() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"defaults", Defaults(), false},
		{"idioma vazio é aceito", Config{PresetID: "custom", MaxParallel: 1}, false},
		{"idioma en", Config{PresetID: "custom", Language: "en", MaxParallel: 1}, false},
		{"idioma em maiúsculo", Config{PresetID: "custom", Language: "PT", MaxParallel: 1}, false},
		{"crf no limite", Config{PresetID: "custom", CRF: MaxCRF, Language: "pt", MaxParallel: 1}, false},
		{"preset em branco", Config{PresetID: "\t", Language: "pt", MaxParallel: 1}, true},
		{"idioma inválido", Config{PresetID: "custom", Language: "de", MaxParallel: 1}, true},
		{"paralelismo inválido", Config{PresetID: "custom", Language: "pt", MaxParallel: 0}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Error("Validate deveria falhar")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate = %v, want nil", err)
			}
		})
	}
}

func TestSaveNormalizesBeforeWriting(t *testing.T) {
	store := newTestStore(t)
	// MaxParallel 0 é inválido e reprovado; mas um valor alto e um idioma em
	// maiúsculas devem ser normalizados no arquivo sem falhar o Save.
	cfg := Config{PresetID: "youtube", Language: "EN", MaxParallel: 4}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("config.json inválido: %v", err)
	}
	if raw["language"] != "en" {
		t.Errorf("language no arquivo = %v, want en normalizado", raw["language"])
	}
}

func TestLoadAppliesDefaultsToAbsentKeys(t *testing.T) {
	store := newTestStore(t)
	if err := os.WriteFile(store.Path(), []byte(`{"presetId":"youtube"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PresetID != "youtube" {
		t.Errorf("PresetID = %q, want youtube", cfg.PresetID)
	}
	if cfg.Language != "pt" {
		t.Errorf("Language = %q, want o default pt", cfg.Language)
	}
	if cfg.MaxParallel != 1 {
		t.Errorf("MaxParallel = %d, want o default 1", cfg.MaxParallel)
	}
	if cfg.SizeMB != 10 {
		t.Errorf("SizeMB = %v, want o default 10", cfg.SizeMB)
	}
}

func TestLoadRepairsOutOfRangeValues(t *testing.T) {
	store := newTestStore(t)
	raw := `{"presetId":"custom","sizeMB":-4,"crf":900,"maxParallel":0,"language":"PT"}`
	if err := os.WriteFile(store.Path(), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SizeMB != 0 {
		t.Errorf("SizeMB = %v, want 0", cfg.SizeMB)
	}
	if cfg.CRF != MaxCRF {
		t.Errorf("CRF = %v, want %v", cfg.CRF, MaxCRF)
	}
	if cfg.MaxParallel != 1 {
		t.Errorf("MaxParallel = %d, want 1", cfg.MaxParallel)
	}
	if cfg.Language != "pt" {
		t.Errorf("Language = %q, want pt", cfg.Language)
	}
}

func TestLoadIgnoresUnknownKeys(t *testing.T) {
	store := newTestStore(t)
	raw := `{"presetId":"shorts","campoDoFuturo":{"a":1},"maxParallel":2}`
	if err := os.WriteFile(store.Path(), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PresetID != "shorts" || cfg.MaxParallel != 2 {
		t.Errorf("Load() = %+v, want os campos conhecidos preservados", cfg)
	}
}

func TestClamp(t *testing.T) {
	cases := []struct {
		name         string
		v, low, high float64
		want         float64
	}{
		{"dentro", 5, 0, 10, 5},
		{"abaixo", -1, 0, 10, 0},
		{"acima", 11, 0, 10, 10},
		{"no limite inferior", 0, 0, 10, 0},
		{"no limite superior", 10, 0, 10, 10},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := clamp(tt.v, tt.low, tt.high); got != tt.want {
				t.Errorf("clamp(%v,%v,%v) = %v, want %v", tt.v, tt.low, tt.high, got, tt.want)
			}
		})
	}
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}
