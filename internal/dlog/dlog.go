// Package dlog emite rastros de diagnóstico no stderr quando o modo local de
// depuração (VIDCTL_DEBUG) está ativo. Sem a variável, cada chamada é um no-op.
package dlog

import (
	"io"
	"log"
	"os"
)

var logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)

// Enabled informa se o modo de depuração foi ligado via VIDCTL_DEBUG.
// Qualquer valor diferente de vazio e de "0" ativa o rastro.
func Enabled() bool {
	v := os.Getenv("VIDCTL_DEBUG")
	return v != "" && v != "0"
}

// Printf registra uma linha de diagnóstico quando o modo está ativo.
func Printf(format string, args ...any) {
	if !Enabled() {
		return
	}
	logger.Printf(format, args...)
}

// SetOutput redireciona a saída do rastro (usado por testes).
func SetOutput(w io.Writer) {
	logger.SetOutput(w)
}
