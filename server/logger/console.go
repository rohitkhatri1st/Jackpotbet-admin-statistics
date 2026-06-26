package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// ConsoleWriter is an io.Writer used for console output.
type ConsoleWriter io.Writer

// NewStandardConsoleWriter returns stdout as a ConsoleWriter.
func NewStandardConsoleWriter() ConsoleWriter {
	return os.Stdout
}

// logFieldsOrder defines the display order for known log fields.
// Fields not listed here appear after these in alphabetical order.
var logFieldsOrder = []string{
	"type",
	"port",
	"requestId",
	"method",
	"path",
	"duration",
	"status",
	"msg",
}

// NewZeroLogConsoleWriter wraps out with zerolog's pretty-print formatter.
// Known fields are printed in logFieldsOrder; any unlisted fields follow alphabetically.
func NewZeroLogConsoleWriter(out io.Writer) io.Writer {
	return zerolog.ConsoleWriter{
		Out:         out,
		FieldsOrder: logFieldsOrder,
	}
}
