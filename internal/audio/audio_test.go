package audio

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	base := Spec{InputPath: "/videos/entrada.mp4", OutputPath: "/out/x.mp3"}
	tests := []struct {
		name     string
		spec     Spec
		contains []string
		absent   []string
	}{
		{
			name:     "mp3 com bitrate",
			spec:     Spec{InputPath: "a.mp4", OutputPath: "o.mp3", Codec: "mp3", BitrateKbps: 192},
			contains: []string{"-c:a", "libmp3lame", "-b:a", "192k"},
		},
		{
			name:   "mp3 sem bitrate usa o default do formato",
			spec:   Spec{InputPath: "a.mp4", OutputPath: "o.mp3", Codec: "mp3"},
			absent: []string{"-b:a"},
		},
		{
			name:     "opus",
			spec:     Spec{InputPath: "a.mp4", OutputPath: "o.opus", Codec: "opus", BitrateKbps: 64},
			contains: []string{"-c:a", "libopus", "-b:a", "64k"},
		},
		{
			name:     "aac",
			spec:     Spec{InputPath: "a.mp4", OutputPath: "o.aac", Codec: "aac", BitrateKbps: 128},
			contains: []string{"-c:a", "aac", "-b:a", "128k"},
		},
		{
			name: "flac lossless sem bitrate",
			spec: Spec{InputPath: "a.mp4", OutputPath: "o.flac", Codec: "flac", BitrateKbps: 999},
			// O 999 não chega lá: o Validate barra bitrate em lossless.
			contains: []string{"-c:a", "flac"},
			absent:   []string{"-b:a"},
		},
		{
			name:     "wav pcm_s16le sem bitrate",
			spec:     Spec{InputPath: "a.mp4", OutputPath: "o.wav", Codec: "wav"},
			contains: []string{"-c:a", "pcm_s16le"},
			absent:   []string{"-b:a"},
		},
		{
			name:     "copy só remux",
			spec:     Spec{InputPath: "a.mp4", OutputPath: "o.mka", Codec: "copy"},
			contains: []string{"-c:a", "copy"},
			absent:   []string{"-b:a"},
		},
		{
			name:     "map da primeira faixa de áudio",
			spec:     base,
			contains: []string{"-vn", "-map", "0:a:0"},
		},
		{
			name:     "progresso do ffmpeg habilitado antes do destino",
			spec:     base,
			contains: []string{"-progress", "pipe:1", "-nostats", "/out/x.mp3"},
		},
		{
			name:     "canais e taxa condicionais",
			spec:     Spec{InputPath: "a.mp4", OutputPath: "o.mp3", Codec: "mp3", Channels: 1, SampleRate: 16000},
			contains: []string{"-ac", "1", "-ar", "16000"},
		},
		{
			name:   "sem canais e taxa quando zero",
			spec:   Spec{InputPath: "a.mp4", OutputPath: "o.mp3", Codec: "mp3"},
			absent: []string{"-ac", "-ar"},
		},
		{
			name:     "trim -ss/-to antes do -i",
			spec:     base.withTrim(60, 120),
			contains: []string{"-ss", "60.000", "-to", "120.000", "-i", "/videos/entrada.mp4"},
		},
		{
			name:   "sem trim quando zero",
			spec:   Spec{InputPath: "a.mp4", OutputPath: "o.mp3", Codec: "mp3", TrimStartSec: 0, TrimEndSec: 0},
			absent: []string{"-ss", "-to"},
		},
		{
			name:     "normalizacao loudnorm",
			spec:     Spec{InputPath: "a.mp4", OutputPath: "o.mp3", Codec: "mp3", Normalize: true},
			contains: []string{"-af", "loudnorm=I=-16:LRA=11:TP=-1.5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildArgs(tt.spec)
			for _, want := range tt.contains {
				if !containsSeq(args, want) {
					t.Errorf("argv %v não contém %q", args, want)
				}
			}
			for _, not := range tt.absent {
				if containsSeq(args, not) {
					t.Errorf("argv %v não deveria conter %q", args, not)
				}
			}
		})
	}
}

// containsSeq verifies that the sequence wanted shows up in args in order.
func containsSeq(args []string, wanted string) bool {
	parts := strings.Split(wanted, " ")
	for i, p := range parts {
		found := false
		for j := range args {
			if args[j] == p {
				if i == len(parts)-1 {
					return true
				}
				args = args[j+1:]
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return false
}

func (s Spec) withTrim(start, end float64) Spec {
	s.TrimStartSec = start
	s.TrimEndSec = end
	return s
}

func TestValidate(t *testing.T) {
	valid := Spec{
		InputPath:   "/tmp/a.mp4",
		Codec:       "mp3",
		BitrateKbps: 192,
		OutputPath:  "/tmp/a-audio.mp3",
	}
	tests := []struct {
		name    string
		spec    Spec
		wantMsg string
	}{
		{"válido", valid, ""},
		{"sem entrada", Spec{}, "escolha um vídeo"},
		{"formato desconhecido", valid.withCodec("ogg"), "formato"},
		{"formato vazio", valid.withCodec(""), "formato"},
		{"bitrate em flac", valid.withCodec("flac").withBitrate(192), "não se aplica"},
		{"bitrate em wav", valid.withCodec("wav").withBitrate(192), "não se aplica"},
		{"bitrate em copy", valid.withCodec("copy").withBitrate(128), "não se aplica"},
		{"bitrate baixo", valid.withBitrate(16), "entre 32 e 320"},
		{"bitrate alto", valid.withBitrate(400), "entre 32 e 320"},
		{"canais inválidos", valid.withChannels(3), "canais"},
		{"canais negativos", valid.withChannels(-1), "canais"},
		{"trim invertido", valid.withTrim(30, 10), "maior que o início"},
		{"trim negativo", valid.withTrim(-1, 0), "negativos"},
		{"fim igual ao início", valid.withTrim(30, 30), "maior que o início"},
		{"normalizar com copy", valid.withCodec("copy").withNormalize(), "copy"},
		{"sem destino", valid.withOutput(""), "destino"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantMsg == "" {
				if err != nil {
					t.Fatalf("Validate = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate deveria falhar")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("mensagem %q não contém %q", err, tt.wantMsg)
			}
		})
	}
}

func (s Spec) withCodec(c string) Spec  { s.Codec = c; return s }
func (s Spec) withBitrate(b int) Spec   { s.BitrateKbps = b; return s }
func (s Spec) withChannels(c int) Spec  { s.Channels = c; return s }
func (s Spec) withNormalize() Spec      { s.Normalize = true; return s }
func (s Spec) withOutput(o string) Spec { s.OutputPath = o; return s }

func TestSuggestOutput(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		codec   string
		suffix  string
		want    string
		wantErr bool
	}{
		{"mp3 com sufixo", "/videos/meu clip.mov", "mp3", "-audio", "meu clip-audio.mp3", false},
		{"sufixo vazio vira default", "/videos/a.mp4", "opus", "", "a-audio.opus", false},
		{"aac", "/videos/a.mkv", "aac", "-worst", "a-worst.aac", false},
		{"flac", "/videos/a.webm", "flac", "-audio", "a-audio.flac", false},
		{"wav", "/videos/a.ts", "wav", "-audio", "a-audio.wav", false},
		{"copy", "/videos/a.mp4", "copy", "-audio", "a-audio.mka", false},
		{"código desconhecido", "/videos/a.mp4", "ogg", "-audio", "", true},
		{"caminho vazio", "", "mp3", "-audio", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SuggestOutput(tt.path, tt.codec, tt.suffix)
			if tt.wantErr {
				if err == nil {
					t.Fatal("esperava erro")
				}
				return
			}
			if err != nil {
				t.Fatalf("SuggestOutput: %v", err)
			}
			if got != tt.want {
				t.Errorf("SuggestOutput = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtension(t *testing.T) {
	for codec, want := range map[string]string{
		"mp3": "mp3", "opus": "opus", "aac": "aac",
		"flac": "flac", "wav": "wav", "copy": "mka",
	} {
		got, err := Extension(codec)
		if err != nil {
			t.Fatalf("Extension(%s): %v", codec, err)
		}
		if got != want {
			t.Errorf("Extension(%s) = %q, want %q", codec, got, want)
		}
	}
	if _, err := Extension("ogg"); err == nil {
		t.Error("Extension(ogg) deveria falhar")
	}
}

func TestValidCodecsStable(t *testing.T) {
	a := strings.Join(ValidCodecs(), ",")
	b := strings.Join(ValidCodecs(), ",")
	if a != b || !strings.Contains(a, "mp3") || !strings.Contains(a, "copy") {
		t.Errorf("lista de codecs instável ou incompleta: %q", a)
	}
}

func TestSuggestOutputJoinsWithDir(t *testing.T) {
	got, err := SuggestOutput("/tmp/x.avi", "flac", "-audio")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != got {
		t.Errorf("SuggestOutput deve devolver só o nome do arquivo, got %q", got)
	}
}
