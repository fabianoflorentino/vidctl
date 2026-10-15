package transcribe

import (
	"context"
	"strconv"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
)

// pcmArgs builds the ffmpeg command that turns the (optionally trimmed) source
// into a 16 kHz mono PCM WAV, the common input every speech engine accepts.
// The input seek flags go before -i for a fast seek.
func pcmArgs(input, out string, start, end float64) []string {
	args := []string{"-y", "-hide_banner", "-nostats"}
	if start > 0 {
		args = append(args, "-ss", strconv.FormatFloat(start, 'f', 3, 64))
	}
	if end > 0 {
		args = append(args, "-to", strconv.FormatFloat(end, 'f', 3, 64))
	}
	args = append(args, "-i", input, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", out)
	return args
}

// extractPCM is indirected so tests can skip the real ffmpeg run.
var extractPCM = func(ctx context.Context, ffmpeg, input, out string, start, end float64) error {
	return cmdutil.CommandContext(ctx, ffmpeg, pcmArgs(input, out, start, end)...).Run()
}
