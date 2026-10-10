package encode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
)

const probeTimeout = 15 * time.Second

// probeFrame is the encode test frame. It must be large enough for every
// hardware encoder: NVENC rejects dimensions below its minimum (reported as
// "Frame Dimension less than the minimum supported value" for 64x64 or 128x128
// on recent chips), so a 256x256 frame is used.
const probeFrame = "256x256"

// EncoderInfo describes a detected hardware backend.
type EncoderInfo struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Codecs []string `json:"codecs"` // probed-ok codecs; empty when unavailable
	Error  string   `json:"error"`  // probe failure detail when Codecs is empty
}

// Availability is the result of backend detection.
type Availability struct {
	Hardware []EncoderInfo `json:"hardware"`
}

// Detector caches which ffmpeg encoders exist and probes which hardware
// backends actually encode on this machine.
type Detector struct {
	mu    sync.Mutex
	avail *Availability
	err   error
}

func ffmpegBinary() (string, error) {
	p, err := cmdutil.Resolve("ffmpeg")
	if err != nil {
		return "", errors.New("ffmpeg não encontrado. Instale o ffmpeg para usar o vidctl.")
	}
	return p, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Availability returns the probed backends, caching the result until Refresh.
func (d *Detector) Availability() (Availability, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.avail != nil {
		return *d.avail, nil
	}
	if d.err != nil {
		return Availability{}, d.err
	}

	bin, err := ffmpegBinary()
	if err != nil {
		d.err = err
		return Availability{}, err
	}

	listed, err := listEncoders(bin)
	if err != nil {
		d.err = err
		return Availability{}, err
	}

	avail := Availability{Hardware: []EncoderInfo{}}
	for _, id := range hardwareIDs {
		if !supportedOnGOOS(id) {
			continue
		}
		h264, h265 := VideoCodec(CodecH264, id), VideoCodec(CodecH265, id)
		if !contains(listed, h264) && !contains(listed, h265) {
			continue
		}
		info := EncoderInfo{ID: id, Label: Label(id), Codecs: []string{}}
		for _, codec := range []string{CodecH264, CodecH265} {
			name := VideoCodec(codec, id)
			if !contains(listed, name) {
				continue
			}
			if err := probeEncoder(bin, name); err != nil {
				if info.Error == "" {
					info.Error = err.Error()
				}
				continue
			}
			info.Codecs = append(info.Codecs, codec)
		}
		avail.Hardware = append(avail.Hardware, info)
	}
	d.avail = &avail
	return avail, nil
}

// Refresh drops the cache so the next Availability call re-detects.
func (d *Detector) Refresh() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.avail, d.err = nil, nil
}

// listEncoders runs `ffmpeg -hide_banner -encoders` and returns the video
// encoder names it advertises.
func listEncoders(bin string) ([]string, error) {
	out, err := exec.Command(bin, "-hide_banner", "-encoders").Output()
	if err != nil {
		return nil, fmt.Errorf("falha ao listar encoders do ffmpeg: %w", err)
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "V") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		names = append(names, fields[1])
	}
	return names, nil
}

// supportedOnGOOS gates backends that only exist on one platform: AMF requires
// the AMD runtime, which ships only on Windows, and VideoToolbox is macOS-only.
// NVENC and QSV can be offered anywhere ffmpeg lists them.
func supportedOnGOOS(hw string) bool {
	switch hw {
	case HWAMF:
		return runtime.GOOS == "windows"
	case HWVideotoolbox:
		return runtime.GOOS == "darwin"
	default:
		return true
	}
}

// probeEncoder runs a real 1-frame encode with the given encoder to confirm it
// works on this machine.
func probeEncoder(bin, encoder string) error {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=black:s=" + probeFrame,
		"-frames:v", "1",
		"-c:v", encoder,
		"-f", "null", "-",
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(msg)
	}
	return nil
}
