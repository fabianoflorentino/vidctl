package transcribe

import (
	"strings"
	"testing"
)

func TestTimestampSRTAndVTT(t *testing.T) {
	cases := []struct {
		sec float64
		srt string
		vtt string
	}{
		{0, "00:00:00,000", "00:00:00.000"},
		{1.5, "00:00:01,500", "00:00:01.500"},
		{61.25, "00:01:01,250", "00:01:01.250"},
		{3661.007, "01:01:01,007", "01:01:01.007"},
		{-3, "00:00:00,000", "00:00:00.000"},
	}
	for _, c := range cases {
		if got := timestampSRT(c.sec); got != c.srt {
			t.Errorf("timestampSRT(%v) = %q; want %q", c.sec, got, c.srt)
		}
		if got := timestampVTT(c.sec); got != c.vtt {
			t.Errorf("timestampVTT(%v) = %q; want %q", c.sec, got, c.vtt)
		}
	}
}

func TestFormatSRTAndText(t *testing.T) {
	segs := []Segment{
		{StartSec: 0, EndSec: 2.5, Text: "  Hello world  "},
		{StartSec: 3, EndSec: 4, Text: ""},
		{StartSec: 6, EndSec: 7, Text: "Bye"},
	}
	got := FormatSRT(segs, false)
	want := "1\n00:00:00,000 --> 00:00:02,500\nHello world\n\n2\n00:00:06,000 --> 00:00:07,000\nBye\n\n"
	if got != want {
		t.Errorf("FormatSRT =\n%q\nwant\n%q", got, want)
	}
	if txt := FormatText(segs); txt != "Hello world Bye\n" {
		t.Errorf("FormatText = %q", txt)
	}
	if txt := FormatText(nil); txt != "" {
		t.Errorf("FormatText(nil) = %q; want empty", txt)
	}
}

func TestFormatVTT(t *testing.T) {
	got := FormatVTT([]Segment{{StartSec: 0, EndSec: 1.25, Text: "Hi"}}, false)
	if !strings.HasPrefix(got, "WEBVTT\n\n") {
		t.Fatalf("VTT must start with the header, got %q", got)
	}
	if !strings.Contains(got, "00:00:00.000 --> 00:00:01.250") {
		t.Errorf("VTT timestamp missing: %q", got)
	}
}

func TestWordTimestamps(t *testing.T) {
	segs := []Segment{{
		StartSec: 0, EndSec: 2, Text: "Hello world",
		Words: []Word{
			{StartSec: 0, EndSec: 1, Text: "Hello"},
			{StartSec: 1, EndSec: 2, Text: "world"},
		},
	}}
	got := FormatSRT(segs, true)
	if strings.Count(got, "-->") != 2 {
		t.Errorf("word timestamps should emit one cue per word, got %q", got)
	}
	segOnly := FormatSRT(segs, false)
	if strings.Count(segOnly, "-->") != 1 {
		t.Errorf("without word timestamps one cue is expected, got %q", segOnly)
	}
}

func TestParseWhisperJSONOffsets(t *testing.T) {
	data := []byte(`{"transcription":[
		{"offsets":{"from":0,"to":2000},"text":" Hello"},
		{"offsets":{"from":2000,"to":4000},"text":" world"}
	]}`)
	segs, text, err := ParseWhisperJSON(data)
	if err != nil {
		t.Fatalf("ParseWhisperJSON: %v", err)
	}
	if len(segs) != 2 || segs[0].StartSec != 0 || segs[1].EndSec != 4 {
		t.Fatalf("unexpected segments: %+v", segs)
	}
	if text != "Hello world" {
		t.Errorf("text = %q", text)
	}
}

func TestParseWhisperJSONTimestampsAndWords(t *testing.T) {
	data := []byte(`{"transcription":[
		{"timestamps":{"from":"00:00:01,000","to":"00:00:02,500"},"text":"hi",
		 "words":[{"word":"hi","start":1,"end":2.5}]}
	]}`)
	segs, _, err := ParseWhisperJSON(data)
	if err != nil {
		t.Fatalf("ParseWhisperJSON: %v", err)
	}
	if len(segs) != 1 || segs[0].StartSec != 1 || segs[0].EndSec != 2.5 {
		t.Fatalf("timestamps not parsed: %+v", segs)
	}
	if len(segs[0].Words) != 1 || segs[0].Words[0].Text != "hi" {
		t.Errorf("words not parsed: %+v", segs[0].Words)
	}
}

func TestParseWhisperJSONCentiseconds(t *testing.T) {
	data := []byte(`{"transcription":[{"t0":100,"t1":250,"text":"x"}]}`)
	segs, _, err := ParseWhisperJSON(data)
	if err != nil {
		t.Fatalf("ParseWhisperJSON: %v", err)
	}
	if segs[0].StartSec != 1 || segs[0].EndSec != 2.5 {
		t.Errorf("t0/t1 centiseconds not converted: %+v", segs[0])
	}
}

func TestParseWhisperJSONInvalid(t *testing.T) {
	if _, _, err := ParseWhisperJSON([]byte("{")); err == nil {
		t.Error("malformed JSON should fail")
	}
}

func TestParseVoskJSONGroupsWords(t *testing.T) {
	data := []byte(`{"result":[
		{"word":"hello","start":0,"end":0.5},
		{"word":"world","start":0.6,"end":1.0},
		{"word":"again","start":5.0,"end":5.5}
	],"text":"hello world again"}`)
	segs, text, err := ParseVoskJSON(data)
	if err != nil {
		t.Fatalf("ParseVoskJSON: %v", err)
	}
	if len(segs) != 2 {
		t.Fatalf("gap should split cues, got %d: %+v", len(segs), segs)
	}
	if segs[0].Text != "hello world" || segs[1].Text != "again" {
		t.Errorf("grouping wrong: %+v", segs)
	}
	if text != "hello world again" {
		t.Errorf("text = %q", text)
	}
}

func TestParseVoskJSONTextOnly(t *testing.T) {
	segs, text, err := ParseVoskJSON([]byte(`{"text":"only text"}`))
	if err != nil {
		t.Fatalf("ParseVoskJSON: %v", err)
	}
	if len(segs) != 1 || text != "only text\n" {
		t.Errorf("text-only output not handled: segs=%+v text=%q", segs, text)
	}
}

func TestParseVoskJSONEmpty(t *testing.T) {
	segs, text, err := ParseVoskJSON([]byte(`{}`))
	if err != nil {
		t.Fatalf("ParseVoskJSON: %v", err)
	}
	if len(segs) != 0 || text != "" {
		t.Errorf("empty output should yield nothing: segs=%+v text=%q", segs, text)
	}
}

func TestGroupWordsMaxChars(t *testing.T) {
	var words []Word
	for i := 0; i < 40; i++ {
		words = append(words, Word{StartSec: float64(i), EndSec: float64(i) + 0.5, Text: "palavra"})
	}
	segs := groupWords(words)
	if len(segs) < 2 {
		t.Errorf("long text should break into multiple cues, got %d", len(segs))
	}
}
