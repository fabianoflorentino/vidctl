package compress

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/fabianoflorentino/vidctl/internal/dlog"
)

func TestTraceExecWritesCommand(t *testing.T) {
	t.Setenv("VIDCTL_DEBUG", "1")
	var buf bytes.Buffer
	dlog.SetOutput(&buf)
	t.Cleanup(func() { dlog.SetOutput(os.Stderr) })

	cmd := cmdutil.CommandContext(context.Background(), "/usr/bin/ffmpeg", "-y", "-i", "in.mp4", "-c:v", "hevc_nvenc", "out.mp4")
	traceExec("job-trace", "encoding", cmd)

	if !strings.Contains(buf.String(), "[compress] job=job-trace stage=encoding exec /usr/bin/ffmpeg -y -i in.mp4 -c:v hevc_nvenc out.mp4") {
		t.Errorf("trace inesperado: %q", buf.String())
	}
}

func TestTraceExecNoopWithoutDebug(t *testing.T) {
	t.Setenv("VIDCTL_DEBUG", "0")
	var buf bytes.Buffer
	dlog.SetOutput(&buf)
	t.Cleanup(func() { dlog.SetOutput(os.Stderr) })

	cmd := cmdutil.CommandContext(context.Background(), "/usr/bin/ffmpeg", "-i", "in.mp4", "out.mp4")
	traceExec("job-trace", "encoding", cmd)

	if buf.Len() != 0 {
		t.Errorf("sem VIDCTL_DEBUG o trace não deveria escrever: %q", buf.String())
	}
}
