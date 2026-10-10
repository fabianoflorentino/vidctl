package presets

import "testing"

func TestByIDKnownPresets(t *testing.T) {
	known := []string{"whatsapp-status", "whatsapp-16", "instagram", "shorts", "youtube", "custom"}
	for _, id := range known {
		t.Run(id, func(t *testing.T) {
			p, ok := ByID(id)
			if !ok {
				t.Fatalf("preset %q should exist", id)
			}
			if p.ID != id {
				t.Errorf("got id %q; want %q", p.ID, id)
			}
		})
	}

	t.Run("unknown", func(t *testing.T) {
		if _, ok := ByID("nao-existe"); ok {
			t.Error("expected unknown preset to return ok=false")
		}
	})
}

func TestListUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range List() {
		if seen[p.ID] {
			t.Errorf("duplicate preset id %q", p.ID)
		}
		seen[p.ID] = true
		if p.Mode != "size" && p.Mode != "crf" {
			t.Errorf("preset %q has invalid mode %q", p.ID, p.Mode)
		}
		if p.Codec != "" && p.Codec != "h264" && p.Codec != "h265" {
			t.Errorf("preset %q has invalid codec %q", p.ID, p.Codec)
		}
		if p.Hardware != "" && p.Hardware != "nvenc" && p.Hardware != "amf" && p.Hardware != "qsv" && p.Hardware != "videotoolbox" {
			t.Errorf("preset %q has invalid hardware %q", p.ID, p.Hardware)
		}
	}
}

func TestByIDDefaultsToSoftwareH264(t *testing.T) {
	for _, id := range []string{"whatsapp-status", "youtube", "custom"} {
		p, ok := ByID(id)
		if !ok {
			t.Fatalf("preset %q should exist", id)
		}
		if p.Codec != "" || p.Hardware != "" {
			t.Errorf("preset %q deveria ser software h264 por padrão, got codec=%q hardware=%q", id, p.Codec, p.Hardware)
		}
	}
}
