// Package config persists the user settings of the app in a JSON file under
// the user configuration directory. Loading never fails on a missing or corrupt
// file: the app must always boot with the defaults.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// FileName is the name of the settings file inside the config directory.
	FileName = "config.json"
	// DirName is the per-app directory created under the user config dir.
	DirName = "vidctl"

	// MinCRF and MaxCRF bound the quality value accepted by libx264.
	MinCRF = 0.0
	MaxCRF = 51.0
	// MaxSizeMB caps the size target offered by the size-based presets.
	MaxSizeMB = 4096.0
)

// ErrInvalidConfig is returned by Save when the config carries values the app
// cannot use. The message names the offending field so the UI can show it.
var ErrInvalidConfig = errors.New("configuração inválida")

// Config holds every setting the app remembers between runs.
//
// OutputDir empty means "next to the input video". FFmpegPath and FFprobePath
// empty mean "look up in the system PATH". Language, MaxParallel,
// NotifyOnDone and OpenFolderOnDone are persisted by this phase and consumed
// by later ones, so a round-trip through the file must preserve them.
type Config struct {
	PresetID         string  `json:"presetId"`
	SizeMB           float64 `json:"sizeMB"`
	CRF              float64 `json:"crf"`
	OutputDir        string  `json:"outputDir"`
	FFmpegPath       string  `json:"ffmpegPath"`
	FFprobePath      string  `json:"ffprobePath"`
	Language         string  `json:"language"`
	MaxParallel      int     `json:"maxParallel"`
	NotifyOnDone     bool    `json:"notifyOnDone"`
	OpenFolderOnDone bool    `json:"openFolderOnDone"`
}

// Defaults returns the configuration used on a fresh install.
func Defaults() Config {
	return Config{
		PresetID:         "whatsapp-status",
		SizeMB:           10,
		CRF:              23,
		Language:         "pt",
		MaxParallel:      1,
		NotifyOnDone:     false,
		OpenFolderOnDone: false,
	}
}

// Dir returns the directory holding the settings file.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar a pasta de configuração: %w", err)
	}
	return filepath.Join(base, DirName), nil
}

// Normalize repairs values outside the supported ranges and fills blank
// fields with their default. It never rejects a config, so a hand-edited file
// cannot stop the app from starting.
func (c Config) Normalize() Config {
	out := c
	out.PresetID = strings.TrimSpace(out.PresetID)
	if out.PresetID == "" {
		out.PresetID = Defaults().PresetID
	}
	out.OutputDir = strings.TrimSpace(out.OutputDir)
	out.FFmpegPath = strings.TrimSpace(out.FFmpegPath)
	out.FFprobePath = strings.TrimSpace(out.FFprobePath)
	out.Language = strings.ToLower(strings.TrimSpace(out.Language))
	if out.Language == "" {
		out.Language = Defaults().Language
	}
	out.SizeMB = clamp(out.SizeMB, 0, MaxSizeMB)
	out.CRF = clamp(out.CRF, MinCRF, MaxCRF)
	if out.MaxParallel < 1 {
		out.MaxParallel = 1
	}
	return out
}

// Validate reports whether the config can be persisted as informed. It is
// stricter than Normalize: it refuses values the UI should correct instead of
// silently clamping them.
func (c Config) Validate() error {
	if strings.TrimSpace(c.PresetID) == "" {
		return fmt.Errorf("%w: presetId vazio", ErrInvalidConfig)
	}
	if c.SizeMB < 0 || c.SizeMB > MaxSizeMB {
		return fmt.Errorf("%w: sizeMB precisa estar entre 0 e %.0f", ErrInvalidConfig, MaxSizeMB)
	}
	if c.CRF < MinCRF || c.CRF > MaxCRF {
		return fmt.Errorf("%w: crf precisa estar entre %.0f e %.0f", ErrInvalidConfig, MinCRF, MaxCRF)
	}
	if c.MaxParallel < 1 {
		return fmt.Errorf("%w: maxParallel precisa ser no mínimo 1", ErrInvalidConfig)
	}
	switch strings.ToLower(strings.TrimSpace(c.Language)) {
	case "", "pt", "en":
	default:
		return fmt.Errorf("%w: language precisa ser pt ou en", ErrInvalidConfig)
	}
	return nil
}

// Store reads and writes the config file for one directory.
type Store struct {
	path string
}

// NewStore returns a Store for dir, or for the default user config directory
// when dir is empty. It performs no I/O: a missing directory only fails on
// Save, which keeps reads on a fresh install working.
func NewStore(dir string) (*Store, error) {
	if dir == "" {
		var err error
		if dir, err = Dir(); err != nil {
			return nil, err
		}
	}
	return &Store{path: filepath.Join(dir, FileName)}, nil
}

// Path returns the absolute path of the settings file.
func (s *Store) Path() string {
	return s.path
}

// Load reads the settings file. A missing, unreadable or malformed file yields
// the defaults and a nil error, so the caller can always boot.
func (s *Store) Load() (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(s.path)
	if err != nil {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Defaults(), nil
	}
	return cfg.Normalize(), nil
}

// Save writes the settings atomically: the JSON goes to a temporary file in the
// same directory and is then renamed over the target, so a crash mid-write
// leaves the previous settings intact. The temp file is removed when the write
// or the rename fails.
func (s *Store) Save(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg.Normalize(), "", "  ")
	if err != nil {
		return fmt.Errorf("não foi possível serializar a configuração: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("não foi possível criar %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, FileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("não foi possível criar o arquivo temporário: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("não foi possível escrever a configuração: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("não foi possível gravar a configuração: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("não foi possível ajustar as permissões: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("não foi possível salvar a configuração: %w", err)
	}
	return nil
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
