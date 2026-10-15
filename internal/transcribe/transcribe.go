package transcribe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/dlog"
	"github.com/fabianoflorentino/vidctl/internal/events"
	"github.com/fabianoflorentino/vidctl/internal/media"
)

// DefaultSuffix is appended to the file name of a transcript.
const DefaultSuffix = "-transcript"

// Spec describes one transcription job coming from the frontend.
type Spec struct {
	InputPath      string  `json:"inputPath"`
	OutputBase     string  `json:"outputBase"` // full path without extension
	Engine         string  `json:"engine"`     // "auto"|"whispercpp"|"vosk"
	Language       string  `json:"language"`   // "auto"|"pt"|"en"
	ModelPath      string  `json:"modelPath"`  // local model file/dir
	Format         string  `json:"format"`     // "all"|"srt"|"vtt"|"txt"
	WordTimestamps bool    `json:"wordTimestamps"`
	TrimStartSec   float64 `json:"trimStartSec"`
	TrimEndSec     float64 `json:"trimEndSec"`
}

// Result lists the generated files and the full plain text.
type Result struct {
	Files []string `json:"files"`
	Text  string   `json:"text"`
}

// ErrInvalidSpec reports a transcription request the app cannot honor.
var ErrInvalidSpec = errors.New("invalid transcription specification")

// ErrNoTranscriber reports that no local speech-to-text engine is installed.
var ErrNoTranscriber = errors.New("no local speech-to-text engine found; install whisper-cli or vosk-transcriber (see the docs)")

// ErrNoModel reports that the engine needs a model path that was not provided.
var ErrNoModel = errors.New("transcription model not configured; set the model path in the preferences")

// Validate refuses requests with values the app cannot use. The messages name
// the offending option so the UI can show the exact problem.
func (s Spec) Validate() error {
	if strings.TrimSpace(s.InputPath) == "" {
		return fmt.Errorf("%w: pick a video to transcribe", ErrInvalidSpec)
	}
	switch e := normalize(s.Engine); e {
	case "", EngineAuto, EngineWhisper, EngineVosk:
	default:
		return fmt.Errorf("%w: unknown engine %q (use %s)", ErrInvalidSpec, s.Engine, strings.Join(SupportedEngines(), ", "))
	}
	switch l := normalize(s.Language); l {
	case "", "auto", "pt", "en":
	default:
		return fmt.Errorf("%w: language must be auto, pt or en", ErrInvalidSpec)
	}
	switch f := normalize(s.Format); f {
	case "", "all", "srt", "vtt", "txt":
	default:
		return fmt.Errorf("%w: format must be all, srt, vtt or txt", ErrInvalidSpec)
	}
	if s.TrimStartSec < 0 || s.TrimEndSec < 0 {
		return fmt.Errorf("%w: transcription times cannot be negative", ErrInvalidSpec)
	}
	if s.TrimEndSec > 0 && s.TrimEndSec <= s.TrimStartSec {
		return fmt.Errorf("%w: the end of the transcription must be greater than the start", ErrInvalidSpec)
	}
	if strings.TrimSpace(s.OutputBase) == "" {
		return fmt.Errorf("%w: choose the transcript destination", ErrInvalidSpec)
	}
	return nil
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// SuggestName returns the transcript file name without extension:
// <name><suffix>. An empty suffix uses DefaultSuffix.
func SuggestName(inputPath, suffix string) (string, error) {
	base := filepath.Base(strings.TrimSpace(inputPath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("%w: provide the source video", ErrInvalidSpec)
	}
	if strings.TrimSpace(suffix) == "" {
		suffix = DefaultSuffix
	}
	return strings.TrimSuffix(base, filepath.Ext(base)) + suffix, nil
}

// ffmpegPath resolves the ffmpeg binary, honouring the configured override. It
// is indirected so tests can avoid the real lookup.
var ffmpegPath = func() (string, error) {
	p, err := cmdutil.Resolve("ffmpeg")
	if errors.Is(err, cmdutil.ErrNotConfigured) {
		return "", err
	}
	if err != nil {
		return "", errors.New("ffmpeg not found. Install ffmpeg to use vidctl.")
	}
	return p, nil
}

// probe is indirected so tests can skip the real ffprobe run.
var probe = media.Probe

// runProc runs a CLI. It is indirected so tests can fake the engine output.
var runProc = func(ctx context.Context, bin string, args []string) error {
	return cmdutil.CommandContext(ctx, bin, args...).Run()
}

// runEngine executes the chosen engine and returns its timed segments.
var runEngine = runEngineDefault

func runEngineDefault(ctx context.Context, engine string, spec Spec, wavPath, workDir string) ([]Segment, error) {
	switch engine {
	case EngineWhisper:
		return runWhisper(ctx, spec, wavPath, workDir)
	case EngineVosk:
		return runVosk(ctx, spec, wavPath, workDir)
	default:
		return nil, ErrNoTranscriber
	}
}

func runWhisper(ctx context.Context, spec Spec, wavPath, workDir string) ([]Segment, error) {
	if strings.TrimSpace(spec.ModelPath) == "" {
		return nil, ErrNoModel
	}
	bin, ok := resolveEngine(EngineWhisper)
	if !ok {
		return nil, ErrNoTranscriber
	}
	base := filepath.Join(workDir, "whisper")
	if err := runProc(ctx, bin, whisperArgs(base, spec, wavPath)); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(base + ".json")
	if err != nil {
		return nil, fmt.Errorf("whisper did not produce a transcript: %w", err)
	}
	segs, _, err := ParseWhisperJSON(data)
	return segs, err
}

func whisperArgs(base string, spec Spec, wavPath string) []string {
	lang := normalize(spec.Language)
	if lang == "" {
		lang = "auto"
	}
	args := []string{"-m", spec.ModelPath, "-f", wavPath, "-oj", "-of", base, "-l", lang}
	if spec.WordTimestamps {
		args = append(args, "-ml", "1")
	}
	return args
}

func runVosk(ctx context.Context, spec Spec, wavPath, workDir string) ([]Segment, error) {
	bin, ok := resolveEngine(EngineVosk)
	if !ok {
		return nil, ErrNoTranscriber
	}
	outJSON := filepath.Join(workDir, "vosk.json")
	if err := runProc(ctx, bin, voskArgs(outJSON, spec, wavPath)); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(outJSON)
	if err != nil {
		return nil, fmt.Errorf("vosk did not produce a transcript: %w", err)
	}
	segs, _, err := ParseVoskJSON(data)
	return segs, err
}

func voskArgs(outJSON string, spec Spec, wavPath string) []string {
	args := []string{"-i", wavPath, "-o", outJSON}
	if m := strings.TrimSpace(spec.ModelPath); m != "" {
		args = append(args, "-m", m)
	}
	if l := normalize(spec.Language); l != "" && l != "auto" {
		args = append(args, "-l", l)
	}
	return args
}

// Transcribe runs one transcription job: it extracts a PCM track, runs the
// chosen engine and writes the requested subtitle/text files, emitting the
// compress:progress/done/error events with Kind "transcribe".
func Transcribe(ctx context.Context, jobID string, spec Spec) (Result, error) {
	fail := func(err error) (Result, error) {
		events.EmitError(jobID, events.KindTranscribe, err.Error())
		return Result{}, err
	}
	if err := spec.Validate(); err != nil {
		return fail(err)
	}

	engine, err := pickEngine(spec.Engine)
	if err != nil {
		return fail(err)
	}
	requested := normalize(spec.Engine)
	if engine == EngineWhisper && strings.TrimSpace(spec.ModelPath) == "" {
		// An explicit whisper request without a model is an error; "auto"
		// falls back to vosk when it is installed.
		if requested != "" && requested != EngineAuto {
			return fail(ErrNoModel)
		}
		if _, ok := resolveEngine(EngineVosk); ok {
			engine = EngineVosk
		} else {
			return fail(ErrNoModel)
		}
	}

	info, err := probe(spec.InputPath)
	if err != nil {
		return fail(err)
	}
	if !info.HasAudio {
		return fail(errors.New("the video has no audio track"))
	}

	ffmpeg, err := ffmpegPath()
	if err != nil {
		return fail(err)
	}

	workDir, err := os.MkdirTemp("", "vidctl-transcribe-*")
	if err != nil {
		return fail(fmt.Errorf("could not create a temporary folder: %w", err))
	}
	defer os.RemoveAll(workDir)
	wavPath := filepath.Join(workDir, "audio.wav")

	events.EmitProgress(jobID, events.KindTranscribe, "extracting", 0)
	if err := extractPCM(ctx, ffmpeg, spec.InputPath, wavPath, spec.TrimStartSec, spec.TrimEndSec); err != nil {
		return fail(err)
	}
	dlog.Printf("[transcribe] job=%s engine=%s wav=%s", jobID, engine, wavPath)

	events.EmitProgress(jobID, events.KindTranscribe, "transcribing", 20)
	segs, err := runEngine(ctx, engine, spec, wavPath, workDir)
	if err != nil {
		return fail(err)
	}

	files, text, sizeBytes, err := writeOutputs(spec, segs)
	if err != nil {
		return fail(err)
	}
	dlog.Printf("[transcribe] job=%s done files=%v", jobID, files)

	primary := spec.OutputBase
	if len(files) > 0 {
		primary = files[0]
	}
	events.EmitDone(jobID, events.KindTranscribe, primary, sizeBytes, float64(sizeBytes)/(1024*1024))
	return Result{Files: files, Text: text}, nil
}

// writeOutputs renders the requested formats next to the output base and
// returns the file list, the plain text and the total size in bytes.
func writeOutputs(spec Spec, segs []Segment) ([]string, string, int64, error) {
	text := FormatText(segs)
	format := normalize(spec.Format)
	if format == "" {
		format = "all"
	}
	var files []string
	var size int64
	write := func(ext, content string) error {
		p := spec.OutputBase + "." + ext
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return err
		}
		files = append(files, p)
		if st, err := os.Stat(p); err == nil {
			size += st.Size()
		}
		return nil
	}

	if format == "all" || format == "srt" {
		if err := write("srt", FormatSRT(segs, spec.WordTimestamps)); err != nil {
			return nil, "", 0, err
		}
	}
	if format == "all" || format == "vtt" {
		if err := write("vtt", FormatVTT(segs, spec.WordTimestamps)); err != nil {
			return nil, "", 0, err
		}
	}
	if format == "all" || format == "txt" {
		if err := write("txt", text); err != nil {
			return nil, "", 0, err
		}
	}
	if len(files) == 0 {
		return nil, "", 0, fmt.Errorf("%w: unknown format %q", ErrInvalidSpec, spec.Format)
	}
	return files, strings.TrimSpace(text), size, nil
}
