// Package media probes media files with ffprobe and exposes their metadata.
package media

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
)

// Info describes an input video probed with ffprobe.
type Info struct {
	Path        string  `json:"path"`
	DurationSec float64 `json:"durationSec"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	HasAudio    bool    `json:"hasAudio"`
	HasVideo    bool    `json:"hasVideo"`
	SizeMB      float64 `json:"sizeMB"`
	VideoCodec  string  `json:"videoCodec"`
	AudioCodec  string  `json:"audioCodec"`
}

// flexFloat accepts both JSON numbers and string-encoded numbers.
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}

type ffprobeOutput struct {
	Streams []struct {
		CodecType string    `json:"codec_type"`
		CodecName string    `json:"codec_name"`
		Width     int       `json:"width"`
		Height    int       `json:"height"`
		Duration  flexFloat `json:"duration"`
		Tags      struct {
			Rotate string `json:"rotate"`
		} `json:"tags"`
	} `json:"streams"`
	Format struct {
		Duration flexFloat `json:"duration"`
	} `json:"format"`
}

func ffprobePath() (string, error) {
	p, err := cmdutil.LookPath("ffprobe")
	if err != nil {
		return "", errors.New("ffprobe não encontrado. Instale o ffmpeg para usar o vidctl.")
	}
	return p, nil
}

// Probe extracts duration, resolution and codecs from a media file.
func Probe(path string) (*Info, error) {
	bin, err := ffprobePath()
	if err != nil {
		return nil, err
	}

	out, err := cmdutil.Command(bin,
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	).Output()
	if err != nil {
		return nil, errors.New("não foi possível ler o arquivo de vídeo: " + path)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, err
	}

	info := &Info{Path: path}

	var audioCodecs []string
	rotation := 0
	for _, s := range probe.Streams {
		switch s.CodecType {
		case "video":
			info.HasVideo = true
			info.VideoCodec = s.CodecName
			if r, err := strconv.Atoi(s.Tags.Rotate); err == nil {
				rotation = r
			}
			w, h := s.Width, s.Height
			if rotation == 90 || rotation == 270 {
				w, h = h, w
			}
			if w > info.Width {
				info.Width = w
			}
			if h > info.Height {
				info.Height = h
			}
		case "audio":
			info.HasAudio = true
			audioCodecs = append(audioCodecs, s.CodecName)
		}
	}
	if len(audioCodecs) > 0 {
		info.AudioCodec = audioCodecs[0]
	}

	info.DurationSec = float64(probe.Format.Duration)
	if info.DurationSec == 0 {
		for _, s := range probe.Streams {
			if s.CodecType == "video" && float64(s.Duration) > 0 {
				info.DurationSec = float64(s.Duration)
				break
			}
		}
	}

	if st, err := os.Stat(path); err == nil {
		info.SizeMB = float64(st.Size()) / (1024 * 1024)
	}

	if !info.HasVideo {
		return nil, errors.New("o arquivo não contém um stream de vídeo")
	}
	if info.DurationSec <= 0 {
		return nil, errors.New("não foi possível determinar a duração do vídeo")
	}

	return info, nil
}
