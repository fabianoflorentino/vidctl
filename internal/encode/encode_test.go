package encode

import (
	"strings"
	"testing"
)

func join(args []string) string { return strings.Join(args, " ") }

func TestVideoCodec(t *testing.T) {
	cases := []struct {
		codec, hw, want string
	}{
		{"", "", "libx264"},
		{CodecH264, "", "libx264"},
		{CodecH265, "", "libx265"},
		{CodecH264, HWNVENC, "h264_nvenc"},
		{CodecH265, HWNVENC, "hevc_nvenc"},
		{CodecH264, HWQSV, "h264_qsv"},
		{CodecH265, HWQSV, "hevc_qsv"},
		{CodecH264, HWAMF, "h264_amf"},
		{CodecH265, HWAMF, "hevc_amf"},
		{CodecH264, HWVideotoolbox, "h264_videotoolbox"},
		{CodecH265, HWVideotoolbox, "hevc_videotoolbox"},
		{CodecH265, Auto, "libx265"},
		{"av1", "", "libx264"},
	}
	for _, c := range cases {
		if got := VideoCodec(c.codec, c.hw); got != c.want {
			t.Errorf("VideoCodec(%q, %q) = %q, want %q", c.codec, c.hw, got, c.want)
		}
	}
}

func TestValidCodecHardware(t *testing.T) {
	for _, c := range []string{"", CodecH264, CodecH265} {
		if !ValidCodec(c) {
			t.Errorf("ValidCodec(%q) = false, want true", c)
		}
	}
	for _, c := range []string{"av1", "h266", "H264"} {
		if ValidCodec(c) {
			t.Errorf("ValidCodec(%q) = true, want false", c)
		}
	}
	for _, h := range []string{"", Auto, HWNVENC, HWQSV, HWAMF, HWVideotoolbox} {
		if !ValidHardware(h) {
			t.Errorf("ValidHardware(%q) = false, want true", h)
		}
	}
	for _, h := range []string{"cuda", "vaapi", "gpu"} {
		if ValidHardware(h) {
			t.Errorf("ValidHardware(%q) = true, want false", h)
		}
	}
}

func TestTwoPass(t *testing.T) {
	cases := []struct {
		hw   string
		want bool
	}{
		{"", true},
		{HWQSV, true},
		{HWNVENC, false},
		{HWAMF, false},
		{HWVideotoolbox, false},
		{Auto, false},
	}
	for _, c := range cases {
		if got := TwoPass(c.hw); got != c.want {
			t.Errorf("TwoPass(%q) = %v, want %v", c.hw, got, c.want)
		}
	}
}

func TestQualityArgs(t *testing.T) {
	cases := []struct {
		codec, hw string
		crf       float64
		wantParts []string
		notWant   []string
	}{
		{CodecH264, "", 23, []string{"-c:v libx264", "-crf 23", "-preset medium"}, nil},
		{CodecH265, "", 23, []string{"-c:v libx265", "-crf 23"}, nil},
		{CodecH264, HWNVENC, 23, []string{"-c:v h264_nvenc", "-cq 23", "-rc vbr"}, nil},
		{CodecH265, HWNVENC, 30, []string{"-c:v hevc_nvenc", "-cq 30"}, nil},
		{CodecH264, HWQSV, 23, []string{"-c:v h264_qsv", "-global_quality 23"}, nil},
		{CodecH264, HWAMF, 23, []string{"-c:v h264_amf", "-rc cqp", "-qp_i 23", "-qp_p 23"}, nil},
		{CodecH264, HWVideotoolbox, 23, []string{"-c:v h264_videotoolbox", "-q:v 23"}, nil},
		{CodecH264, HWVideotoolbox, 0, []string{"-q:v 1"}, nil},
		{CodecH264, HWVideotoolbox, 120, []string{"-q:v 100"}, nil},
	}
	for _, c := range cases {
		got := join(QualityArgs(c.codec, c.hw, c.crf))
		for _, want := range c.wantParts {
			if !strings.Contains(got, want) {
				t.Errorf("QualityArgs(%q,%q,%v) = %q, falta %q", c.codec, c.hw, c.crf, got, want)
			}
		}
		for _, nw := range c.notWant {
			if strings.Contains(got, nw) {
				t.Errorf("QualityArgs(%q,%q,%v) = %q, não deveria conter %q", c.codec, c.hw, c.crf, got, nw)
			}
		}
	}
}

func TestSizeArgs(t *testing.T) {
	got := join(SizeArgs(CodecH264, "", 1000000, 1100000, 2200000))
	for _, want := range []string{"-c:v libx264", "-b:v 1000000", "-maxrate 1100000", "-bufsize 2200000", "-preset medium"} {
		if !strings.Contains(got, want) {
			t.Errorf("software size args = %q, falta %q", got, want)
		}
	}
	nvenc := join(SizeArgs(CodecH265, HWNVENC, 500000, 550000, 1100000))
	for _, want := range []string{"-c:v hevc_nvenc", "-rc vbr", "-b:v 500000", "-maxrate 550000"} {
		if !strings.Contains(nvenc, want) {
			t.Errorf("nvenc size args = %q, falta %q", nvenc, want)
		}
	}
	if strings.Contains(nvenc, "-crf") {
		t.Errorf("nvenc size args não deve conter -crf: %q", nvenc)
	}
	qsv := join(SizeArgs(CodecH264, HWQSV, 500000, 550000, 1100000))
	if !strings.Contains(qsv, "-c:v h264_qsv") || strings.Contains(qsv, "-preset") {
		t.Errorf("qsv size args inesperadas: %q", qsv)
	}
	amf := join(SizeArgs(CodecH264, HWAMF, 500000, 550000, 1100000))
	if !strings.Contains(amf, "-rc vbr") || !strings.Contains(amf, "-quality quality") {
		t.Errorf("amf size args inesperadas: %q", amf)
	}
	vt := join(SizeArgs(CodecH264, HWVideotoolbox, 500000, 550000, 1100000))
	if !strings.Contains(vt, "-c:v h264_videotoolbox") || !strings.Contains(vt, "-b:v 500000") {
		t.Errorf("videotoolbox size args inesperadas: %q", vt)
	}
}

func TestPass1Args(t *testing.T) {
	sw := Pass1Args(CodecH264, "", 1000)
	if !strings.Contains(join(sw), "-preset medium") {
		t.Errorf("software pass1 deve incluir preset: %v", sw)
	}
	qsv := Pass1Args(CodecH264, HWQSV, 1000)
	if strings.Contains(join(qsv), "-preset") {
		t.Errorf("qsv pass1 não deve incluir preset: %v", qsv)
	}
}

func TestResolveAuto(t *testing.T) {
	cases := []struct {
		avail []string
		want  string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{CodecH264}, ""},
		{[]string{HWVideotoolbox}, HWVideotoolbox},
		{[]string{HWAMF, HWNVENC}, HWNVENC},
		{[]string{HWQSV, HWAMF}, HWQSV},
	}
	for _, c := range cases {
		if got := ResolveAuto(c.avail); got != c.want {
			t.Errorf("ResolveAuto(%v) = %q, want %q", c.avail, got, c.want)
		}
	}
}

func TestLabel(t *testing.T) {
	if Label(HWNVENC) != "NVIDIA NVENC" {
		t.Errorf("Label(nvenc) = %q", Label(HWNVENC))
	}
	if Label("") != "Software (CPU)" {
		t.Errorf("Label(\"\") = %q", Label(""))
	}
}
