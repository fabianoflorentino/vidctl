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
var ErrInvalidConfig = errors.New("invalid configuration")

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
	// AudioSuffix is appended to the file name of an extracted track.
	AudioSuffix string `json:"audioSuffix"`
	// Transcribe* persist the phase-12 transcription preferences.
	TranscribeEngine         string `json:"transcribeEngine"`
	TranscribeModelPath      string `json:"transcribeModelPath"`
	TranscribeLanguage       string `json:"transcribeLanguage"`
	TranscribeWordTimestamps bool   `json:"transcribeWordTimestamps"`
	TranscribeFormat         string `json:"transcribeFormat"`
	// TranscribeSuffix is appended to the file name of a transcript.
	TranscribeSuffix string `json:"transcribeSuffix"`
}

// Defaults returns the configuration used on a fresh install.
func Defaults() Config {
	return Config{
		PresetID:           "whatsapp-status",
		SizeMB:             10,
		CRF:                23,
		Language:           "pt",
		MaxParallel:        1,
		NotifyOnDone:       false,
		OpenFolderOnDone:   false,
		AudioSuffix:        "-audio",
		TranscribeEngine:   "auto",
		TranscribeLanguage: "auto",
		TranscribeFormat:   "all",
		TranscribeSuffix:   "-transcript",
	}
}

// Dir returns the directory holding the settings file.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not locate the configuration folder: %w", err)
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
	out.AudioSuffix = strings.TrimSpace(out.AudioSuffix)
	if out.AudioSuffix == "" {
		out.AudioSuffix = Defaults().AudioSuffix
	}
	out.TranscribeEngine = strings.ToLower(strings.TrimSpace(out.TranscribeEngine))
	if !validTranscribeEngine(out.TranscribeEngine) {
		out.TranscribeEngine = Defaults().TranscribeEngine
	}
	out.TranscribeModelPath = strings.TrimSpace(out.TranscribeModelPath)
	out.TranscribeLanguage = strings.ToLower(strings.TrimSpace(out.TranscribeLanguage))
	if out.TranscribeLanguage == "" {
		out.TranscribeLanguage = Defaults().TranscribeLanguage
	}
	out.TranscribeFormat = strings.ToLower(strings.TrimSpace(out.TranscribeFormat))
	if !validTranscribeFormat(out.TranscribeFormat) {
		out.TranscribeFormat = Defaults().TranscribeFormat
	}
	out.TranscribeSuffix = strings.TrimSpace(out.TranscribeSuffix)
	if out.TranscribeSuffix == "" {
		out.TranscribeSuffix = Defaults().TranscribeSuffix
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
		return fmt.Errorf("%w: empty presetId", ErrInvalidConfig)
	}
	if c.SizeMB < 0 || c.SizeMB > MaxSizeMB {
		return fmt.Errorf("%w: sizeMB must be between 0 and %.0f", ErrInvalidConfig, MaxSizeMB)
	}
	if c.CRF < MinCRF || c.CRF > MaxCRF {
		return fmt.Errorf("%w: crf must be between %.0f and %.0f", ErrInvalidConfig, MinCRF, MaxCRF)
	}
	if c.MaxParallel < 1 {
		return fmt.Errorf("%w: maxParallel must be at least 1", ErrInvalidConfig)
	}
	switch strings.ToLower(strings.TrimSpace(c.Language)) {
	case "", "pt", "en", "es", "zh":
	default:
		return fmt.Errorf("%w: language must be pt, en, es or zh", ErrInvalidConfig)
	}
	if e := strings.ToLower(strings.TrimSpace(c.TranscribeEngine)); e != "" && !validTranscribeEngine(e) {
		return fmt.Errorf("%w: transcribeEngine must be auto, whispercpp or vosk", ErrInvalidConfig)
	}
	if f := strings.ToLower(strings.TrimSpace(c.TranscribeFormat)); f != "" && !validTranscribeFormat(f) {
		return fmt.Errorf("%w: transcribeFormat must be all, srt, vtt or txt", ErrInvalidConfig)
	}
	return nil
}

// validTranscribeEngine reports whether id is a recognized transcription engine.
func validTranscribeEngine(id string) bool {
	switch id {
	case "auto", "whispercpp", "vosk":
		return true
	default:
		return false
	}
}

// validTranscribeFormat reports whether id is a recognized transcript format.
func validTranscribeFormat(id string) bool {
	switch id {
	case "all", "srt", "vtt", "txt":
		return true
	default:
		return false
	}
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
		return fmt.Errorf("could not serialize the configuration: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("could not create %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, FileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("could not create the temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("could not write the configuration: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not write the configuration: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("could not set permissions: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("could not save the configuration: %w", err)
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
