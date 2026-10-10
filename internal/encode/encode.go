// Package encode builds ffmpeg video-encoder arguments for the supported
// codecs (h264/h265) and hardware backends (software, NVENC, qsv, amf,
// VideoToolbox) and detects which encoders are available at runtime.
package encode

import (
	"math"
	"strconv"
)

const (
	CodecH264 = "h264"
	CodecH265 = "h265"
)

// Hardware backend ids. "" selects the software encoders (libx264/libx265).
const (
	HWNVENC        = "nvenc"
	HWQSV          = "qsv"
	HWAMF          = "amf"
	HWVideotoolbox = "videotoolbox"
	Auto           = "auto"
)

// SoftwarePreset is the ffmpeg -preset used for libx264/libx265.
const SoftwarePreset = "medium"

// HWNVENCPreset balances speed and quality for the NVIDIA encoders.
const HWNVENCPreset = "p4"

// nvencPresetIDs are the accepted NVENC quality presets, p1 (fastest) to p7
// (best quality).
var nvencPresetIDs = map[string]bool{
	"p1": true, "p2": true, "p3": true, "p4": true, "p5": true, "p6": true, "p7": true,
}

// NvencPreset normalizes a user-chosen NVENC preset id, falling back to the
// balanced default when empty or unknown.
func NvencPreset(p string) string {
	if nvencPresetIDs[p] {
		return p
	}
	return HWNVENCPreset
}

// ValidNvencPreset reports whether p is a recognized NVENC preset id.
func ValidNvencPreset(p string) bool { return nvencPresetIDs[p] }

var (
	// hardwareIDs is the supported backends in auto-resolution priority order.
	hardwareIDs = []string{HWNVENC, HWQSV, HWAMF, HWVideotoolbox}

	validHW = map[string]bool{
		"": true, Auto: true,
		HWNVENC: true, HWQSV: true, HWAMF: true, HWVideotoolbox: true,
	}
)

// ValidCodec reports whether c is a known codec id ("" means h264).
func ValidCodec(c string) bool {
	return c == "" || c == CodecH264 || c == CodecH265
}

// ValidHardware reports whether h is a known hardware backend id.
func ValidHardware(h string) bool { return validHW[h] }

// VideoCodec returns the ffmpeg -c:v encoder name for the codec/backend pair.
// Unknown backends (e.g. Auto before resolution) fall back to software.
func VideoCodec(codec, hw string) string {
	if hw == HWNVENC {
		if codec == CodecH265 {
			return "hevc_nvenc"
		}
		return "h264_nvenc"
	}
	if hw == HWQSV {
		if codec == CodecH265 {
			return "hevc_qsv"
		}
		return "h264_qsv"
	}
	if hw == HWAMF {
		if codec == CodecH265 {
			return "hevc_amf"
		}
		return "h264_amf"
	}
	if hw == HWVideotoolbox {
		if codec == CodecH265 {
			return "hevc_videotoolbox"
		}
		return "h264_videotoolbox"
	}
	if codec == CodecH265 {
		return "libx265"
	}
	return "libx264"
}

// TwoPass reports whether size-limited encoding uses the two-pass pipeline for
// hw. Software (x264/x265) and qsv support two-pass; the other backends use a
// single VBR pass to limit the size.
func TwoPass(hw string) bool {
	return hw == "" || hw == HWQSV
}

// QualityArgs returns the -c:v section for constant-quality encoding.
func QualityArgs(codec, hw string, crf float64) []string {
	return QualityArgsP(codec, hw, crf, "")
}

// QualityArgsP is QualityArgs with an optional NVENC preset override (p1..p7).
func QualityArgsP(codec, hw string, crf float64, nvencPreset string) []string {
	n := int(math.Round(crf))
	enc := VideoCodec(codec, hw)
	switch hw {
	case HWNVENC:
		return []string{"-c:v", enc, "-preset", NvencPreset(nvencPreset), "-rc", "vbr", "-cq", strconv.Itoa(n), "-b:v", "0"}
	case HWQSV:
		return []string{"-c:v", enc, "-global_quality", strconv.Itoa(n)}
	case HWAMF:
		return []string{"-c:v", enc, "-quality", "quality", "-rc", "cqp", "-qp_i", strconv.Itoa(n), "-qp_p", strconv.Itoa(n)}
	case HWVideotoolbox:
		return []string{"-c:v", enc, "-q:v", strconv.Itoa(vtQuality(crf))}
	default:
		return []string{"-c:v", enc, "-crf", strconv.Itoa(n), "-preset", SoftwarePreset}
	}
}

// SizeArgs returns the -c:v section for size-limited VBR encoding.
func SizeArgs(codec, hw string, vbr, maxrate, bufsize int) []string {
	return SizeArgsP(codec, hw, vbr, maxrate, bufsize, "")
}

// SizeArgsP is SizeArgs with an optional NVENC preset override (p1..p7).
func SizeArgsP(codec, hw string, vbr, maxrate, bufsize int, nvencPreset string) []string {
	enc := VideoCodec(codec, hw)
	vbrS, maxS, bufS := strconv.Itoa(vbr), strconv.Itoa(maxrate), strconv.Itoa(bufsize)
	switch hw {
	case HWNVENC:
		return []string{"-c:v", enc, "-preset", NvencPreset(nvencPreset), "-rc", "vbr", "-b:v", vbrS, "-maxrate", maxS, "-bufsize", bufS}
	case HWQSV:
		return []string{"-c:v", enc, "-b:v", vbrS, "-maxrate", maxS, "-bufsize", bufS}
	case HWAMF:
		return []string{"-c:v", enc, "-quality", "quality", "-rc", "vbr", "-b:v", vbrS, "-maxrate", maxS, "-bufsize", bufS}
	case HWVideotoolbox:
		return []string{"-c:v", enc, "-b:v", vbrS, "-maxrate", maxS, "-bufsize", bufS}
	default:
		return []string{"-c:v", enc, "-b:v", vbrS, "-maxrate", maxS, "-bufsize", bufS, "-preset", SoftwarePreset}
	}
}

// Pass1Args returns the -c:v section used by the first (analysis) pass of
// two-pass size-limited encoding.
func Pass1Args(codec, hw string, vbr int) []string {
	enc := VideoCodec(codec, hw)
	args := []string{"-c:v", enc, "-b:v", strconv.Itoa(vbr)}
	if hw != HWQSV {
		args = append(args, "-preset", SoftwarePreset)
	}
	return args
}

// vtQuality maps the CRF scale to the VideoToolbox -q:v scale (1-100), keeping
// the same direction (lower = better). This is a heuristic; VideoToolbox tunes
// quality per frame and may still overshoot the target.
func vtQuality(crf float64) int {
	q := int(math.Round(crf))
	if q < 1 {
		q = 1
	}
	if q > 100 {
		q = 100
	}
	return q
}

// ResolveAuto maps the Auto sentinel to the first available backend in
// priority order, or "" (software) when none is available.
func ResolveAuto(available []string) string {
	for _, id := range hardwareIDs {
		for _, a := range available {
			if a == id {
				return id
			}
		}
	}
	return ""
}

// Label returns a display name for a hardware backend id.
func Label(hw string) string {
	switch hw {
	case HWNVENC:
		return "NVIDIA NVENC"
	case HWQSV:
		return "Intel Quick Sync (qsv)"
	case HWAMF:
		return "AMD AMF"
	case HWVideotoolbox:
		return "Apple VideoToolbox"
	default:
		return "Software (CPU)"
	}
}
