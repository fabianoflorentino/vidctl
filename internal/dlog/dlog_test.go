package dlog

import (
	"bytes"
	"strings"
	"testing"
)

func capture(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := logger.Writer()
	SetOutput(&buf)
	t.Cleanup(func() { SetOutput(prev) })
	return &buf
}

func TestEnabled(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want bool
	}{
		{name: "ligado", env: "1", want: true},
		{name: "ligado com outro valor", env: "true", want: true},
		{name: "desligado explicito", env: "0", want: false},
		{name: "vazio", env: "", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VIDCTL_DEBUG", tc.env)
			if got := Enabled(); got != tc.want {
				t.Fatalf("Enabled() = %v, quero %v", got, tc.want)
			}
		})
	}
}

func TestPrintfSemSaidaQuandoDesabilitado(t *testing.T) {
	t.Setenv("VIDCTL_DEBUG", "")
	buf := capture(t)
	Printf("nao deve aparecer %d", 42)
	if buf.Len() != 0 {
		t.Fatalf("Printf escreveu com o modo desligado: %q", buf.String())
	}
}

func TestPrintfRegistraQuandoAtivo(t *testing.T) {
	t.Setenv("VIDCTL_DEBUG", "1")
	buf := capture(t)
	Printf("job=%s pos=%d", "abc", 3)
	got := buf.String()
	if !strings.Contains(got, "job=abc") || !strings.Contains(got, "pos=3") {
		t.Fatalf("Printf nao registrou a linha: %q", got)
	}
}

func TestPrintfDesligadoPorZero(t *testing.T) {
	t.Setenv("VIDCTL_DEBUG", "0")
	buf := capture(t)
	Printf("silencio")
	if buf.Len() != 0 {
		t.Fatalf("Printf escreveu com VIDCTL_DEBUG=0: %q", buf.String())
	}
}
