// Package presets defines the target platform profiles used to build
// the ffmpeg command.
package presets

// Preset defines a target platform profile.
type Preset struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Mode         string  `json:"mode"`         // "size" targets a max file size (2-pass) | "crf" targets quality (CRF)
	SizeMB       float64 `json:"sizeMB"`       // used when Mode == "size"
	CRF          float64 `json:"crf"`          // used when Mode == "crf"
	AudioBitrate string  `json:"audioBitrate"` // e.g. "96k"
}

var presets = []Preset{
	{
		ID:           "whatsapp-status",
		Name:         "WhatsApp Status",
		Description:  "Até 10MB — ideal para status e conversas",
		Mode:         "size",
		SizeMB:       10,
		CRF:          23,
		AudioBitrate: "96k",
	},
	{
		ID:           "whatsapp-16",
		Name:         "WhatsApp (até 16MB)",
		Description:  "Tamanho maior para vídeos mais longos",
		Mode:         "size",
		SizeMB:       16,
		CRF:          23,
		AudioBitrate: "96k",
	},
	{
		ID:           "instagram",
		Name:         "Instagram Reels / Stories",
		Description:  "Até 16MB — qualidade preservada para vertical",
		Mode:         "size",
		SizeMB:       16,
		CRF:          23,
		AudioBitrate: "96k",
	},
	{
		ID:           "shorts",
		Name:         "YouTube Shorts",
		Description:  "Até 30MB — vertical 9:16",
		Mode:         "size",
		SizeMB:       30,
		CRF:          23,
		AudioBitrate: "96k",
	},
	{
		ID:           "youtube",
		Name:         "YouTube (vídeo normal)",
		Description:  "Qualidade em primeiro lugar, sem limite de tamanho",
		Mode:         "crf",
		SizeMB:       0,
		CRF:          23,
		AudioBitrate: "128k",
	},
	{
		ID:           "custom",
		Name:         "Tamanho personalizado",
		Description:  "Você define o tamanho máximo em MB",
		Mode:         "size",
		SizeMB:       10,
		CRF:          23,
		AudioBitrate: "96k",
	},
}

// List returns all available presets.
func List() []Preset {
	return presets
}

// ByID returns a copy of the preset with the given ID.
func ByID(id string) (Preset, bool) {
	for _, p := range presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}
