package encode

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
)

func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell bins não executam no Windows")
	}
}

func fakeFFmpeg(t *testing.T, encoders, probeFail string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ffmpeg")
	script := `#!/bin/sh
	if [ "$2" = "-encoders" ]; then
		printf '%s' "$FAKE_ENC_LIST"
		exit 0
	fi
	if [ -n "$FAKE_PROBE_FAIL" ] && case "$*" in *"$FAKE_PROBE_FAIL"*) true ;; *) false ;; esac; then
		echo "encoder indisponível no probe" >&2
		exit 1
	fi
	exit 0
	`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_ENC_LIST", encoders)
	t.Setenv("FAKE_PROBE_FAIL", probeFail)
	cmdutil.SetOverride("ffmpeg", path)
	t.Cleanup(cmdutil.ClearOverrides)
	return path
}

const encodersList = ` V....D libx264              libx264 H.264 / AVC / MPEG-4 AVC codec
 V..... h264_nvenc           NVIDIA NVENC H.264 encoder (codec h264)
 V..... hevc_nvenc           NVIDIA NVENC HEVC encoder (codec hevc)
 V..... h264_qsv             Intel Quick Sync H.264 encoder (codec h264)
 V..... hevc_qsv             Intel Quick Sync HEVC encoder (codec hevc)
 V..... h264_videotoolbox    VideoToolbox H.264 Encoder (codec h264)
 V..... hevc_videotoolbox    VideoToolbox HEVC Encoder (codec hevc)
 A..... aac                  AAC (Advanced Audio Coding)
`

// amfList advertises only the AMD backends so the whole backend can be probed
// with a forced failure, leaving Codecs empty.
const amfList = ` V..... h264_amf             AMD AMF H.264 Encoder (codec h264)
 V..... hevc_amf             AMD AMF HEVC Encoder (codec hevc)
`

func TestDetectorAvailability(t *testing.T) {
	skipOnWindows(t)
	fakeFFmpeg(t, encodersList, "hevc_videotoolbox")

	d := &Detector{}
	avail, err := d.Availability()
	if err != nil {
		t.Fatalf("Availability() error: %v", err)
	}

	nvenc := findInfo(avail, HWNVENC)
	if nvenc == nil || !sameStrings(nvenc.Codecs, []string{CodecH264, CodecH265}) {
		t.Errorf("nvenc esperava codecs [h264 h265], got %+v", nvenc)
	}
	qsv := findInfo(avail, HWQSV)
	if qsv == nil || !sameStrings(qsv.Codecs, []string{CodecH264, CodecH265}) {
		t.Errorf("qsv esperava codecs [h264 h265], got %+v", qsv)
	}
	vt := findInfo(avail, HWVideotoolbox)
	if vt == nil || !sameStrings(vt.Codecs, []string{CodecH264}) || vt.Error == "" {
		t.Errorf("videotoolbox esperava apenas h264 com erro de probe, got %+v", vt)
	}
	if findInfo(avail, HWAMF) != nil {
		t.Errorf("amf não deveria aparecer, got %+v", avail.Hardware)
	}
}

func TestDetectorCachesAndRefreshes(t *testing.T) {
	skipOnWindows(t)
	path := fakeFFmpeg(t, encodersList, "hevc_videotoolbox")

	d := &Detector{}
	first, err := d.Availability()
	if err != nil {
		t.Fatalf("Availability() #1 error: %v", err)
	}
	second, err := d.Availability()
	if err != nil {
		t.Fatalf("Availability() #2 error: %v", err)
	}
	// Caching: both calls point at the same cached result.
	if first.Hardware == nil || len(first.Hardware) != len(second.Hardware) {
		t.Fatalf("cache inesperado: %+v vs %+v", first.Hardware, second.Hardware)
	}
	if len(first.Hardware) != 3 {
		t.Fatalf("esperava 3 backends, got %d", len(first.Hardware))
	}

	// Backends desapareceram: Refresh deve re-detectar.
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	d.Refresh()
	after, err := d.Availability()
	if err != nil {
		t.Fatalf("Availability() pós-refresh error: %v", err)
	}
	if len(after.Hardware) != 0 {
		t.Errorf("após refresh esperava 0 backends, got %+v", after.Hardware)
	}
}

func TestDetectorFFmpegMissing(t *testing.T) {
	d := &Detector{}
	cmdutil.SetOverride("ffmpeg", filepath.Join(t.TempDir(), "inexistente"))
	t.Cleanup(cmdutil.ClearOverrides)
	if _, err := d.Availability(); err == nil {
		t.Fatal("esperava erro com ffmpeg ausente")
	}
}

func TestDetectorListParsesOnlyVideo(t *testing.T) {
	skipOnWindows(t)
	fakeFFmpeg(t, encodersList, "")
	names, err := listEncoders(mustResolve(t))
	if err != nil {
		t.Fatalf("listEncoders error: %v", err)
	}
	if !contains(names, "libx264") || !contains(names, "h264_nvenc") {
		t.Errorf("esperava libx264/h264_nvenc na lista, got %v", names)
	}
	if contains(names, "aac") {
		t.Errorf("encoders de áudio não devem entrar na lista, got %v", names)
	}
}

func TestProbeEncoderFailure(t *testing.T) {
	skipOnWindows(t)
	fakeFFmpeg(t, encodersList, "h264_qsv")
	err := probeEncoder(mustResolve(t), "h264_qsv")
	if err == nil {
		t.Fatal("esperava erro ao forçar probe falho")
	}
	if !strings.Contains(err.Error(), "indisponível") {
		t.Errorf("erro deveria carregar a mensagem do stderr, got %v", err)
	}
}

func TestDetectorBackendWithoutCodecs(t *testing.T) {
	skipOnWindows(t)
	fakeFFmpeg(t, amfList, "amf")
	d := &Detector{}
	avail, err := d.Availability()
	if err != nil {
		t.Fatalf("Availability() error: %v", err)
	}
	amf := findInfo(avail, HWAMF)
	if amf == nil {
		t.Fatalf("amf deveria aparecer listado mesmo com probe falho, got %+v", avail.Hardware)
	}
	if amf.Codecs == nil {
		t.Error("codecs não pode ser nil: viraria null no JSON e quebraria o frontend")
	}
	if len(amf.Codecs) != 0 {
		t.Errorf("codecs deveria estar vazio, got %v", amf.Codecs)
	}
	if amf.Error == "" {
		t.Error("erro do probe deveria ser preservado para a UI")
	}
}

func mustResolve(t *testing.T) string {
	t.Helper()
	p, err := cmdutil.Resolve("ffmpeg")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return p
}

func findInfo(a Availability, id string) *EncoderInfo {
	for i := range a.Hardware {
		if a.Hardware[i].ID == id {
			return &a.Hardware[i]
		}
	}
	return nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
