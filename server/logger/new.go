package logger

import (
	"io"
	"time"

	"github.com/rs/zerolog"
)

// NewLogger returns a Logger writing to cw (console) and/or fw (file).
// Pass nil for any destination you don't need.
func NewLogger(cw, fw io.Writer) Logger {
	var writers []io.Writer

	if cw != nil {
		writers = append(writers, cw)
	}
	if fw != nil {
		writers = append(writers, fw)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	mw := io.MultiWriter(writers...)
	zlog := zerolog.New(mw).With().Timestamp().CallerWithSkipFrameCount(3).Logger()
	return &zerologLogger{log: zlog}
}

// zerologLogger is the zerolog-backed implementation of Logger.
type zerologLogger struct {
	log zerolog.Logger
}

func (l *zerologLogger) Debug(fields ...any) {
	addFields(l.log.Debug(), fields...).Msg("")
}

func (l *zerologLogger) Info(fields ...any) {
	addFields(l.log.Info(), fields...).Msg("")
}

func (l *zerologLogger) Warn(fields ...any) {
	addFields(l.log.Warn(), fields...).Msg("")
}

func (l *zerologLogger) Error(err error, fields ...any) {
	addFields(l.log.Error().Err(err), fields...).Msg("")
}

func (l *zerologLogger) With(fields ...any) Logger {
	ctx := l.log.With()
	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			ctx = appendContextField(ctx, "msg", fields[i])
			break
		}
		key, ok := fields[i].(string)
		if !ok {
			key = "msg"
		}
		ctx = appendContextField(ctx, key, fields[i+1])
	}
	return &zerologLogger{log: ctx.Logger()}
}

func appendContextField(ctx zerolog.Context, key string, value any) zerolog.Context {
	switch v := value.(type) {
	case string:
		return ctx.Str(key, v)
	case int:
		return ctx.Int(key, v)
	case int64:
		return ctx.Int64(key, v)
	case float64:
		return ctx.Float64(key, v)
	case bool:
		return ctx.Bool(key, v)
	case time.Time:
		return ctx.Time(key, v)
	case time.Duration:
		return ctx.Dur(key, v)
	case error:
		return ctx.AnErr(key, v)
	default:
		return ctx.Interface(key, v)
	}
}

func addFields(e *zerolog.Event, fields ...any) *zerolog.Event {
	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			// Trailing value with no key — default key to "msg"
			e = appendField(e, "msg", fields[i])
			break
		}
		key, ok := fields[i].(string)
		if !ok {
			key = "msg"
		}
		e = appendField(e, key, fields[i+1])
	}
	return e
}

func appendField(e *zerolog.Event, key string, value any) *zerolog.Event {
	switch v := value.(type) {
	case string:
		return e.Str(key, v)
	case int:
		return e.Int(key, v)
	case int64:
		return e.Int64(key, v)
	case float64:
		return e.Float64(key, v)
	case bool:
		return e.Bool(key, v)
	case time.Time:
		return e.Time(key, v)
	case time.Duration:
		return e.Dur(key, v)
	case error:
		return e.AnErr(key, v)
	default:
		return e.Interface(key, v)
	}
}
