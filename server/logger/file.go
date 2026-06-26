package logger

import (
	"io"
	"os"
	"path/filepath"
)

// NewFileWriter opens (or creates) a log file at path/fileName for appending.
// Returns nil if the file cannot be opened — caller should check before use.
func NewFileWriter(fileName, path string) io.Writer {
	fullPath := filepath.Join(path, fileName)
	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	return f
}
