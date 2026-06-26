package logger

import (
	"context"
	"io"
	"log/slog"
	"runtime"
	"time"
)

// NewSLogger returns a slog-backed Logger writing to cw (console) and/or fw (file).
// Pass nil for any destination you don't need.
func NewSLogger(cw, fw io.Writer) Logger {
	var writers []io.Writer
	if cw != nil {
		writers = append(writers, cw)
	}
	if fw != nil {
		writers = append(writers, fw)
	}

	w := io.Discard
	if len(writers) > 0 {
		w = io.MultiWriter(writers...)
	}

	h := slog.NewTextHandler(w, &slog.HandlerOptions{AddSource: true})
	return &slogLogger{log: slog.New(h)}
}

type slogLogger struct {
	log *slog.Logger
}

func (l *slogLogger) Debug(fields ...any) {
	l.emit(slog.LevelDebug, nil, fields)
}

func (l *slogLogger) Info(fields ...any) {
	l.emit(slog.LevelInfo, nil, fields)
}

func (l *slogLogger) Warn(fields ...any) {
	l.emit(slog.LevelWarn, nil, fields)
}

func (l *slogLogger) Error(err error, fields ...any) {
	l.emit(slog.LevelError, err, fields)
}

func (l *slogLogger) With(fields ...any) Logger {
	return &slogLogger{log: l.log.With(normalizeFields(fields)...)}
}

// emit builds a slog.Record with the correct caller PC so source points to
// the actual call site, not the wrapper. Skip breakdown:
//   - 0: runtime.Callers
//   - 1: emit
//   - 2: Debug / Info / Warn / Error
//   - 3: caller (user code)
func (l *slogLogger) emit(level slog.Level, err error, fields []any) {
	if !l.log.Enabled(context.Background(), level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, "", pcs[0])
	if err != nil {
		r.Add("error", err.Error())
	}
	r.Add(normalizeFields(fields)...)
	_ = l.log.Handler().Handle(context.Background(), r)
}

// normalizeFields converts a trailing unpaired value to ("msg", value)
// so log.Info("server started") logs as msg=server started.
func normalizeFields(fields []any) []any {
	if len(fields)%2 == 0 {
		return fields
	}
	result := make([]any, len(fields)+1)
	copy(result, fields[:len(fields)-1])
	result[len(fields)-1] = "msg"
	result[len(fields)] = fields[len(fields)-1]
	return result
}
