package transcribe

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Word is one recognized token with its own timing.
type Word struct {
	StartSec float64 `json:"startSec"`
	EndSec   float64 `json:"endSec"`
	Text     string  `json:"text"`
}

// Segment is a recognized block with a start and end time. Words carries
// per-token timings when the engine provides them.
type Segment struct {
	StartSec float64 `json:"startSec"`
	EndSec   float64 `json:"endSec"`
	Text     string  `json:"text"`
	Words    []Word  `json:"words,omitempty"`
}

// timestampSRT renders seconds as "HH:MM:SS,mmm".
func timestampSRT(sec float64) string {
	h, m, s, ms := splitTime(sec)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// timestampVTT renders seconds as "HH:MM:SS.mmm".
func timestampVTT(sec float64) string {
	h, m, s, ms := splitTime(sec)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

func splitTime(sec float64) (h, m, s, ms int) {
	if sec < 0 {
		sec = 0
	}
	total := int64(sec*1000 + 0.5)
	ms = int(total % 1000)
	total /= 1000
	s = int(total % 60)
	total /= 60
	m = int(total % 60)
	h = int(total / 60)
	return h, m, s, ms
}

// cues flattens segments into the list of (start, end, text) lines a subtitle
// file needs. With word timestamps it emits one cue per word; otherwise one cue
// per segment.
func cues(segs []Segment, wordTimestamps bool) [][3]any {
	var out [][3]any
	for _, seg := range segs {
		text := strings.TrimSpace(seg.Text)
		if wordTimestamps && len(seg.Words) > 0 {
			for _, w := range seg.Words {
				wt := strings.TrimSpace(w.Text)
				if wt == "" {
					continue
				}
				out = append(out, [3]any{w.StartSec, w.EndSec, wt})
			}
			continue
		}
		if text == "" {
			continue
		}
		out = append(out, [3]any{seg.StartSec, seg.EndSec, text})
	}
	return out
}

// FormatSRT renders segments as a SubRip subtitle file.
func FormatSRT(segs []Segment, wordTimestamps bool) string {
	var b strings.Builder
	for i, c := range cues(segs, wordTimestamps) {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, timestampSRT(c[0].(float64)), timestampSRT(c[1].(float64)), c[2].(string))
	}
	return b.String()
}

// FormatVTT renders segments as a WebVTT subtitle file.
func FormatVTT(segs []Segment, wordTimestamps bool) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for i, c := range cues(segs, wordTimestamps) {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, timestampVTT(c[0].(float64)), timestampVTT(c[1].(float64)), c[2].(string))
	}
	return b.String()
}

// FormatText joins the segment text into a plain paragraph.
func FormatText(segs []Segment) string {
	parts := make([]string, 0, len(segs))
	for _, seg := range segs {
		if t := strings.TrimSpace(seg.Text); t != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ") + "\n"
}

// whisperJSON is the shape of whisper.cpp's JSON output (-oj).
type whisperJSON struct {
	Transcription []struct {
		Timestamps struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"timestamps"`
		Offsets struct {
			From float64 `json:"from"`
			To   float64 `json:"to"`
		} `json:"offsets"`
		T0    float64 `json:"t0"`
		T1    float64 `json:"t1"`
		Text  string  `json:"text"`
		Words []struct {
			Word  string  `json:"word"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"words"`
	} `json:"transcription"`
}

// ParseWhisperJSON converts whisper.cpp JSON into timed segments.
func ParseWhisperJSON(data []byte) ([]Segment, string, error) {
	var doc whisperJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, "", fmt.Errorf("could not read the whisper output: %w", err)
	}
	var segs []Segment
	for _, item := range doc.Transcription {
		start, end := whisperTimes(item.Offsets.From, item.Offsets.To, item.Timestamps.From, item.Timestamps.To, item.T0, item.T1)
		seg := Segment{StartSec: start, EndSec: end, Text: strings.TrimSpace(item.Text)}
		for _, w := range item.Words {
			seg.Words = append(seg.Words, Word{StartSec: w.Start, EndSec: w.End, Text: strings.TrimSpace(w.Word)})
		}
		segs = append(segs, seg)
	}
	return segs, joinText(segs), nil
}

// whisperTimes prefers millisecond offsets, then the "HH:MM:SS,mmm" strings,
// then the centisecond t0/t1 fields older builds emit.
func whisperTimes(offFrom, offTo float64, tsFrom, tsTo string, t0, t1 float64) (float64, float64) {
	if offTo > 0 {
		return offFrom / 1000, offTo / 1000
	}
	if s, ok := parseClock(tsFrom); ok {
		e, _ := parseClock(tsTo)
		return s, e
	}
	if t1 > 0 {
		return t0 / 100, t1 / 100
	}
	return 0, 0
}

// parseClock parses "HH:MM:SS,mmm" or "HH:MM:SS.mmm" into seconds.
func parseClock(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, false
	}
	h, err1 := strconv.ParseFloat(parts[0], 64)
	m, err2 := strconv.ParseFloat(parts[1], 64)
	sec, err3 := strconv.ParseFloat(parts[2], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, false
	}
	return h*3600 + m*60 + sec, true
}

// voskJSON is the shape of vosk's JSON output.
type voskJSON struct {
	Text   string `json:"text"`
	Result []struct {
		Word  string  `json:"word"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"result"`
}

// ParseVoskJSON converts vosk JSON into timed segments. Vosk reports one entry
// per word; consecutive words are grouped into readable cues.
func ParseVoskJSON(data []byte) ([]Segment, string, error) {
	var doc voskJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, "", fmt.Errorf("could not read the vosk output: %w", err)
	}
	var words []Word
	for _, r := range doc.Result {
		t := strings.TrimSpace(r.Word)
		if t == "" {
			continue
		}
		words = append(words, Word{StartSec: r.Start, EndSec: r.End, Text: t})
	}
	if len(words) == 0 {
		text := strings.TrimSpace(doc.Text)
		if text == "" {
			return nil, "", nil
		}
		return []Segment{{Text: text}}, text + "\n", nil
	}
	segs := groupWords(words)
	return segs, joinText(segs), nil
}

// groupWords packs words into cues, breaking on a silence gap or when a cue
// grows too long.
func groupWords(words []Word) []Segment {
	const (
		maxGapSec  = 1.0
		maxCueChar = 80
	)
	var segs []Segment
	var cur Segment
	flush := func() {
		if strings.TrimSpace(cur.Text) != "" {
			segs = append(segs, cur)
		}
		cur = Segment{}
	}
	for i, w := range words {
		if i > 0 {
			gap := w.StartSec - words[i-1].EndSec
			if gap > maxGapSec || len([]rune(cur.Text))+len([]rune(w.Text))+1 > maxCueChar {
				flush()
			}
		}
		if cur.Text == "" {
			cur.StartSec = w.StartSec
		}
		if cur.Text != "" {
			cur.Text += " "
		}
		cur.Text += w.Text
		cur.EndSec = w.EndSec
		cur.Words = append(cur.Words, w)
	}
	flush()
	return segs
}

// joinText concatenates the segment text into a single paragraph.
func joinText(segs []Segment) string {
	parts := make([]string, 0, len(segs))
	for _, seg := range segs {
		if t := strings.TrimSpace(seg.Text); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}
