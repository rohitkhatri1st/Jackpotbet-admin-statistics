package logger

import (
	"io"
	"os"
)

// NewForceLogger returns a logger that always emits to stderr regardless of config,
// plus any additional writer passed in. Delegates to NewLogger for consistent setup.
func NewForceLogger(w io.Writer) Logger {
	return NewLogger(NewZeroLogConsoleWriter(os.Stderr), w)
}
