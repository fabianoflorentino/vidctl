// Package dlog emits diagnostic traces to stderr when the local debug mode
// (VIDCTL_DEBUG) is active. Without the variable, every call is a no-op.
package dlog

import (
	"io"
	"log"
	"os"
)

var logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)

// Enabled reports whether debug mode is on via VIDCTL_DEBUG.
// Qualquer valor diferente de vazio e de "0" ativa o rastro.
func Enabled() bool {
	v := os.Getenv("VIDCTL_DEBUG")
	return v != "" && v != "0"
}

// Printf writes a diagnostic line when debug mode is active.
func Printf(format string, args ...any) {
	if !Enabled() {
		return
	}
	logger.Printf(format, args...)
}

// SetOutput redirects the trace output (used by tests).
func SetOutput(w io.Writer) {
	logger.SetOutput(w)
}
