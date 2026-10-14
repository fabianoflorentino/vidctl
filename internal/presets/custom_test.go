package presets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func validPreset() Preset {
	return Preset{Name: "My WhatsApp", Mode: "size", SizeMB: 12, AudioBitrate: "96k", Codec: "h265"}
}

func TestCustomStoreRoundtrip(t *testing.T) {
	s := newStore(t)
	SetCustomSource(s.LoadAll)
	t.Cleanup(func() { SetCustomSource(nil) })

	got, err := s.Add(validPreset())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(got) != 1 || !strings.HasPrefix(got[0].ID, CustomIDPrefix) {
		t.Fatalf("Add result = %+v; want one custom- preset", got)
	}

	reloaded := s.LoadAll()
	if len(reloaded) != 1 || reloaded[0].Name != "My WhatsApp" || reloaded[0].ID != got[0].ID {
		t.Fatalf("LoadAll = %+v; want the saved preset", reloaded)
	}

	target, _ := ByID(got[0].ID)
	if target.Name != "My WhatsApp" {
		t.Errorf("ByID(custom) = %+v; want the custom preset", target)
	}
}

func TestCustomStoreUpdateAndDelete(t *testing.T) {
	s := newStore(t)
	added, err := s.Add(validPreset())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	id := added[0].ID

	p := validPreset()
	p.Name = "Renamed"
	updated, err := s.Update(id, p)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated[0].Name != "Renamed" || updated[0].ID != id {
		t.Errorf("Update kept %+v; want renamed with same id", updated[0])
	}

	if _, err := s.Delete(CustomIDPrefix + "nope"); err == nil {
		t.Error("Delete of unknown id should fail")
	}
	if _, err := s.Update("whatsapp-status", p); err == nil {
		t.Error("Update of a built-in should fail")
	}
	if _, err := s.Delete("youtube"); err == nil {
		t.Error("Delete of a built-in should fail")
	}
	left, err := s.Delete(id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(left) != 0 {
		t.Errorf("Delete left %+v; want empty", left)
	}
	if _, ok := ByID(id); ok {
		t.Error("deleted preset should no longer resolve")
	}
}

func TestCustomPresetValidation(t *testing.T) {
	bad := []Preset{
		{Name: "", Mode: "size", SizeMB: 12},
		{Name: "x", Mode: "bogus", SizeMB: 12},
		{Name: "x", Mode: "size", SizeMB: 0},
		{Name: "x", Mode: "crf", CRF: 0},
		{Name: "x", Mode: "crf", CRF: 52},
		{Name: "x", Mode: "size", SizeMB: 12, AudioBitrate: "banana"},
		{Name: "x", Mode: "size", SizeMB: 12, Codec: "vp9"},
		{Name: "x", Mode: "size", SizeMB: 12, Hardware: "intel"},
	}
	for _, p := range bad {
		if err := Validate(p); err == nil {
			t.Errorf("Validate(%+v) should fail", p)
		}
	}
	if err := Validate(validPreset()); err != nil {
		t.Errorf("Validate(valid) = %v; want nil", err)
	}
}

func TestCustomStoreUniquenessAndNormalization(t *testing.T) {
	s := newStore(t)
	a, err := s.Add(validPreset())
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Add(validPreset())
	if err != nil {
		t.Fatal(err)
	}
	if a[0].ID != CustomIDPrefix+"my-whatsapp" {
		t.Errorf("first id = %q; want %smy-whatsapp", a[0].ID, CustomIDPrefix)
	}
	if last := b[len(b)-1]; last.ID != CustomIDPrefix+"my-whatsapp-2" {
		t.Errorf("second id = %q; want %smy-whatsapp-2", last.ID, CustomIDPrefix)
	}

	p := validPreset()
	p.Name = "  Camada X  "
	p.Mode = "size"
	p.SizeMB = 9
	p.CRF = 23 // should be dropped on normalize
	p.AudioBitrate = "96K"
	got, err := s.Add(p)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	added := got[len(got)-1]
	if added.Name != "Camada X" || added.CRF != 0 || added.AudioBitrate != "96k" {
		t.Errorf("normalized preset = %+v", added)
	}
}

func TestListMergesCustomsViaSource(t *testing.T) {
	s := newStore(t)
	if _, err := s.Add(validPreset()); err != nil {
		t.Fatal(err)
	}
	SetCustomSource(s.LoadAll)
	t.Cleanup(func() { SetCustomSource(nil) })

	all := List()
	if len(all) != len(presets)+1 {
		t.Fatalf("List() = %d presets; want %d + 1 custom", len(all), len(presets))
	}
	if all[len(all)-1].Name != "My WhatsApp" {
		t.Errorf("custom should come last, got %+v", all[len(all)-1])
	}
	if _, ok := ByID(all[len(all)-1].ID); !ok {
		t.Error("ByID should resolve the custom preset")
	}
}

func TestNewStoreWithoutConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("APPDATA", "")
	if _, err := NewStore(""); err == nil {
		t.Error("NewStore without any config dir should fail")
	}
}

func TestCustomStoreCorruptFileSilent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.LoadAll(); len(got) != 0 {
		t.Errorf("LoadAll on corrupt file = %+v; want empty", got)
	}
}

func TestValidateErrorsAreTyped(t *testing.T) {
	err := Validate(Preset{Name: "x", Mode: "crf", CRF: 0})
	if err == nil || !errors.Is(err, err) {
		t.Fatal("expected a validation error")
	}
}
