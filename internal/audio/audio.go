// Package audio extracts the audio track of a video without re-encoding the
// video, offering explicit format, bitrate, channel and loudness options.
package audio

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/dlog"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
)

// DefaultSuffix is appended to the file name of an extracted track.
const DefaultSuffix = "-audio"

// Spec describes one extraction job coming from the frontend.
type Spec struct {
	InputPath    string  `json:"inputPath"`
	Codec        string  `json:"codec"`        // "mp3"|"opus"|"aac"|"flac"|"wav"|"copy"
	BitrateKbps  int     `json:"bitrateKbps"`  // 0 keeps the format default
	SampleRate   int     `json:"sampleRate"`   // 0 keeps the source rate
	Channels     int     `json:"channels"`     // 0 keeps, 1 mono, 2 stereo
	TrimStartSec float64 `json:"trimStartSec"` // window reused from fase 5
	TrimEndSec   float64 `json:"trimEndSec"`
	Normalize    bool    `json:"normalize"` // loudnorm EBU R128, single-pass
	OutputPath   string  `json:"outputPath"`
}

// ErrInvalidSpec reports an extraction request the app cannot honor.
var ErrInvalidSpec = errors.New("especificação de áudio inválida")

// codecSpec describes how ffmpeg encodes one output format.
type codecSpec struct {
	encoder   string
	bitrate   int // default kbps for lossy formats; unused for lossless
	lossless  bool
	extension string
}

var codecs = map[string]codecSpec{
	"mp3":  {encoder: "libmp3lame", bitrate: 192, extension: "mp3"},
	"opus": {encoder: "libopus", bitrate: 128, extension: "opus"},
	"aac":  {encoder: "aac", bitrate: 160, extension: "aac"},
	"flac": {encoder: "flac", lossless: true, extension: "flac"},
	"wav":  {encoder: "pcm_s16le", lossless: true, extension: "wav"},
	"copy": {encoder: "copy", lossless: true, extension: "mka"},
}

var codecOrder = []string{"mp3", "opus", "aac", "flac", "wav", "copy"}

// ValidCodecs returns the accepted format ids in a stable order.
func ValidCodecs() []string {
	return append([]string(nil), codecOrder...)
}

// Extension returns the file extension (without the dot) for a format.
func Extension(codec string) (string, error) {
	cs, ok := codecs[normalizeCodec(codec)]
	if !ok {
		return "", fmt.Errorf("%w: formato %q desconhecido (use %s)", ErrInvalidSpec, codec, strings.Join(ValidCodecs(), ", "))
	}
	return cs.extension, nil
}

// SuggestOutput builds the output file name in the same folder as the input:
// <name>-<suffix>.<ext>. An empty suffix uses DefaultSuffix.
func SuggestOutput(inputPath, codec, suffix string) (string, error) {
	ext, err := Extension(codec)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(suffix) == "" {
		suffix = DefaultSuffix
	}
	base := filepath.Base(strings.TrimSpace(inputPath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("%w: informe o vídeo de origem", ErrInvalidSpec)
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return base + suffix + "." + ext, nil
}

// Validate refuses extraction requests with values the app cannot use. The
// messages name the offending option so the UI can show the exact problem.
func (s Spec) Validate() error {
	if strings.TrimSpace(s.InputPath) == "" {
		return fmt.Errorf("%w: escolha um vídeo para extrair o áudio", ErrInvalidSpec)
	}
	codec := normalizeCodec(s.Codec)
	cs, ok := codecs[codec]
	if !ok {
		return fmt.Errorf("%w: formato %q desconhecido (use %s)", ErrInvalidSpec, s.Codec, strings.Join(ValidCodecs(), ", "))
	}
	if s.BitrateKbps > 0 && cs.lossless {
		return fmt.Errorf("%w: bitrate não se aplica a %s (perda nenhuma)", ErrInvalidSpec, codec)
	}
	if s.BitrateKbps < 0 || (s.BitrateKbps > 0 && (s.BitrateKbps < 32 || s.BitrateKbps > 320)) {
		return fmt.Errorf("%w: bitrate precisa estar entre 32 e 320 kbps (ou 0 para o padrão)", ErrInvalidSpec)
	}
	if s.SampleRate < 0 {
		return fmt.Errorf("%w: taxa de amostragem inválida", ErrInvalidSpec)
	}
	if s.Channels < 0 || s.Channels > 2 {
		return fmt.Errorf("%w: canais precisam ser 0 (manter), 1 (mono) ou 2 (estéreo)", ErrInvalidSpec)
	}
	if s.TrimStartSec < 0 || s.TrimEndSec < 0 {
		return fmt.Errorf("%w: os tempos de extração não podem ser negativos", ErrInvalidSpec)
	}
	if s.TrimEndSec > 0 && s.TrimEndSec <= s.TrimStartSec {
		return fmt.Errorf("%w: o fim da extração deve ser maior que o início", ErrInvalidSpec)
	}
	if s.Normalize && codec == "copy" {
		return fmt.Errorf("%w: normalizar exige reencodar e não pode ser combinado com copy", ErrInvalidSpec)
	}
	if strings.TrimSpace(s.OutputPath) == "" {
		return fmt.Errorf("%w: escolha o destino do áudio", ErrInvalidSpec)
	}
	return nil
}

func normalizeCodec(codec string) string {
	return strings.ToLower(strings.TrimSpace(codec))
}

// ffmpegPath resolves the ffmpeg binary, honouring the configured override.
func ffmpegPath() (string, error) {
	p, err := cmdutil.Resolve("ffmpeg")
	if errors.Is(err, cmdutil.ErrNotConfigured) {
		return "", err
	}
	if err != nil {
		return "", errors.New("ffmpeg não encontrado. Instale o ffmpeg para usar o vidctl.")
	}
	return p, nil
}

func formatSec(sec float64) string {
	return strconv.FormatFloat(sec, 'f', 3, 64)
}

// buildArgs assembles the ffmpeg command line for the spec. The input seek
// flags (-ss/-to) go before -i for a fast seek, mirroring the compression trim.
func buildArgs(spec Spec) []string {
	cs := codecs[normalizeCodec(spec.Codec)]
	args := []string{"-y"}
	if spec.TrimStartSec > 0 {
		args = append(args, "-ss", formatSec(spec.TrimStartSec))
	}
	if spec.TrimEndSec > 0 {
		args = append(args, "-to", formatSec(spec.TrimEndSec))
	}
	args = append(args, "-i", spec.InputPath)
	args = append(args, "-vn", "-map", "0:a:0")
	args = append(args, "-c:a", cs.encoder)
	if !cs.lossless && spec.BitrateKbps > 0 {
		args = append(args, "-b:a", fmt.Sprintf("%dk", spec.BitrateKbps))
	}
	if cs.encoder != "copy" && spec.SampleRate > 0 {
		args = append(args, "-ar", strconv.Itoa(spec.SampleRate))
	}
	if cs.encoder != "copy" && spec.Channels > 0 {
		args = append(args, "-ac", strconv.Itoa(spec.Channels))
	}
	if spec.Normalize {
		args = append(args, "-af", "loudnorm=I=-16:LRA=11:TP=-1.5")
	}
	args = append(args, "-progress", "pipe:1", "-nostats")
	args = append(args, spec.OutputPath)
	return args
}

// execute runs ffmpeg, scanning `-progress pipe:1` and emitting progress events.
func execute(ctx context.Context, cmd *exec.Cmd, jobID, stage string, duration float64) error {
	dlog.Printf("[audio] job=%s stage=%s exec %s", jobID, stage, strings.Join(cmd.Args, " "))
	progressPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(progressPipe)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			_ = cmd.Process.Kill()
			return err
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") {
			us, _ := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_us="), 64)
			pct := math.Min(us/(duration*1_000_000), 0.999)
			events.EmitProgress(jobID, events.KindAudio, stage, pct*100)
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return err
	}
	return nil
}

// Extract probes the input, validates the request and runs the ffmpeg
// extraction, emitting the same event family the compression queue consumes
// (compress:progress/done/error with Kind "audio"). It returns the output size.
func Extract(ctx context.Context, jobID, inputPath string, spec Spec) (int64, error) {
	fail := func(err error) (int64, error) {
		events.EmitError(jobID, events.KindAudio, err.Error())
		return 0, err
	}
	if err := spec.Validate(); err != nil {
		return fail(err)
	}

	info, err := media.Probe(inputPath)
	if err != nil {
		return fail(err)
	}
	if !info.HasAudio {
		return fail(errors.New("o vídeo não tem faixa de áudio"))
	}
	if inputPath == spec.OutputPath {
		return fail(errors.New("o arquivo de saída não pode ser igual ao de entrada"))
	}

	bin, err := ffmpegPath()
	if err != nil {
		return fail(err)
	}
	if err := os.MkdirAll(filepath.Dir(spec.OutputPath), 0o755); err != nil {
		return fail(fmt.Errorf("falha ao criar pasta de saída: %w", err))
	}

	end := spec.TrimEndSec
	if end <= 0 || end > info.DurationSec {
		end = info.DurationSec
	}
	duration := info.DurationSec
	if spec.TrimStartSec > 0 || spec.TrimEndSec > 0 {
		duration = end - spec.TrimStartSec
	}

	events.EmitProgress(jobID, events.KindAudio, "extracting", 0)
	if err := execute(ctx, cmdutil.CommandContext(ctx, bin, buildArgs(spec)...), jobID, "extracting", duration); err != nil {
		return fail(err)
	}

	var sizeBytes int64
	if st, err := os.Stat(spec.OutputPath); err == nil {
		sizeBytes = st.Size()
	}
	sizeMB := float64(sizeBytes) / (1024 * 1024)
	events.EmitDone(jobID, events.KindAudio, spec.OutputPath, sizeBytes, sizeMB)
	return sizeBytes, nil
}
