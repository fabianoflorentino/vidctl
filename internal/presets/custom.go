package presets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/fabianoflorentino/vidctl/internal/config"
)

const (
	// FileName is the settings file holding the user-created presets.
	FileName = "presets.json"
	// CustomIDPrefix guarantees custom IDs never collide with the built-ins.
	CustomIDPrefix = "custom-"
)

// ErrNoStore is returned when there is no writable configuration folder.
var ErrNoStore = errors.New("no configuration folder available to save presets")

var audioBitrateRE = regexp.MustCompile(`^\d+(\.\d+)?[km]?$`)

// Validate checks a (custom) preset before persisting it. Error messages stay
// in English: the frontend maps them to the UI dictionary.
func Validate(p Preset) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("preset needs a name")
	}
	switch p.Mode {
	case "size":
		if p.SizeMB <= 0 {
			return errors.New("size mode needs a target above 0 MB")
		}
	case "crf":
		if p.CRF <= 0 || p.CRF > 51 {
			return errors.New("crf must be between 0 and 51")
		}
	default:
		return fmt.Errorf("unknown preset mode %q", p.Mode)
	}
	br := strings.ToLower(strings.TrimSpace(p.AudioBitrate))
	if br != "" && !audioBitrateRE.MatchString(br) {
		return fmt.Errorf("invalid audio bitrate %q", p.AudioBitrate)
	}
	switch p.Codec {
	case "", "h264", "h265":
	default:
		return fmt.Errorf("unknown video codec %q", p.Codec)
	}
	switch p.Hardware {
	case "", "nvenc", "amf", "qsv", "videotoolbox":
	default:
		return fmt.Errorf("unknown hardware encoder %q", p.Hardware)
	}
	return nil
}

// Store persists the user-created presets in presets.json, written atomically.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore returns a Store for dir, or the app configuration dir when dir is
// empty. It performs no I/O until a read or write happens.
func NewStore(dir string) (*Store, error) {
	if dir == "" {
		var err error
		if dir, err = config.Dir(); err != nil {
			return nil, fmt.Errorf("could not locate the configuration folder: %w", err)
		}
	}
	return &Store{path: filepath.Join(dir, FileName)}, nil
}

// Path returns the absolute path of the presets file.
func (s *Store) Path() string {
	return s.path
}

// LoadAll returns every saved custom preset. A missing or unreadable file
// yields an empty list, never an error the app must surface.
func (s *Store) LoadAll() []Preset {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.load()
	if err != nil {
		return nil
	}
	return items
}

// Add validates and appends a new custom preset, generating a unique ID. It
// returns the saved custom list after a successful write.
func (s *Store) Add(p Preset) ([]Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := Validate(p); err != nil {
		return nil, err
	}
	items, err := s.load()
	if err != nil {
		return nil, err
	}
	p.ID = newCustomID(p.Name, items)
	items = append(items, normalize(p))
	if err := s.save(items); err != nil {
		return nil, err
	}
	return clone(items), nil
}

// Update replaces the custom preset with the given ID, keeping its identity.
// Built-in presets cannot be edited.
func (s *Store) Update(id string, p Preset) ([]Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !strings.HasPrefix(id, CustomIDPrefix) {
		return nil, errors.New("built-in presets cannot be edited")
	}
	if err := Validate(p); err != nil {
		return nil, err
	}
	items, err := s.load()
	if err != nil {
		return nil, err
	}
	idx := -1
	for i := range items {
		if items[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, errors.New("custom preset not found")
	}
	p.ID = id
	items[idx] = normalize(p)
	if err := s.save(items); err != nil {
		return nil, err
	}
	return clone(items), nil
}

// Delete removes a custom preset. Built-in presets cannot be deleted.
func (s *Store) Delete(id string) ([]Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !strings.HasPrefix(id, CustomIDPrefix) {
		return nil, errors.New("built-in presets cannot be deleted")
	}
	items, err := s.load()
	if err != nil {
		return nil, err
	}
	kept := items[:0]
	found := false
	for _, p := range items {
		if p.ID == id {
			found = true
			continue
		}
		kept = append(kept, p)
	}
	if !found {
		return nil, errors.New("custom preset not found")
	}
	if err := s.save(kept); err != nil {
		return nil, err
	}
	return clone(kept), nil
}

// load reads the presets file without taking the lock (callers hold it).
func (s *Store) load() ([]Preset, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var items []Preset
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// save writes the presets atomically: a temp file is renamed over the target
// so a crash mid-write keeps the previous state. Callers hold the lock.
func (s *Store) save(items []Preset) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("could not serialize the presets: %w", err)
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
		return fmt.Errorf("could not write the presets: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not write the presets: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("could not set permissions: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("could not save the presets: %w", err)
	}
	return nil
}

func clone(items []Preset) []Preset {
	out := append([]Preset(nil), items...)
	return out
}

// normalize trims the free-text fields and drops the mode that is not used.
func normalize(p Preset) Preset {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.AudioBitrate = strings.ToLower(strings.TrimSpace(p.AudioBitrate))
	if p.Mode == "size" {
		p.CRF = 0
	}
	if p.Mode == "crf" {
		p.SizeMB = 0
	}
	return p
}

// newCustomID builds a unique "custom-<slug>" ID derived from the name. The
// slug keeps ascii letters, digits and hyphens; collisions with built-ins or
// saved presets get a numeric suffix.
func newCustomID(name string, existing []Preset) string {
	base := slug(name)
	if base == "" {
		base = "preset"
	}
	taken := map[string]bool{}
	for _, e := range existing {
		taken[e.ID] = true
	}
	id := CustomIDPrefix + base
	for i := 2; taken[id]; i++ {
		id = fmt.Sprintf("%s%s-%d", CustomIDPrefix, base, i)
	}
	return id
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == ' ':
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteByte('-')
			}
		case unicode.IsLetter(r):
			// accented letters fold to their ascii base when possible
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

// customSource is the optional provider of the saved user presets, installed
// by the app so List/ByID transparently include them.
var (
	customMu  sync.RWMutex
	customSrc func() []Preset
)

// SetCustomSource registers the callback that returns the current saved
// presets. The app wires it to its store; passing nil disables custom presets.
func SetCustomSource(fn func() []Preset) {
	customMu.Lock()
	customSrc = fn
	customMu.Unlock()
}

func customPresets() []Preset {
	customMu.RLock()
	fn := customSrc
	customMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn()
}
