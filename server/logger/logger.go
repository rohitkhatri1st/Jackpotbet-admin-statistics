package logger

// Logger is the swappable logging interface. To replace the underlying
// implementation, write a new struct that satisfies this interface and
// swap the constructor in New / NewForce — no call sites change.
type Logger interface {
	// Debug logs a debug-level message.
	// Pass key-value pairs: Debug("key", val, "key2", val2)
	// A single value with no key defaults to the "msg" key: Debug("starting up")
	Debug(fields ...any)

	// Info logs an info-level message.
	// Pass key-value pairs: Info("key", val, "key2", val2)
	// A single value with no key defaults to the "msg" key: Info("request received")
	Info(fields ...any)

	// Warn logs a warning-level message.
	// Pass key-value pairs: Warn("key", val, "key2", val2)
	// A single value with no key defaults to the "msg" key: Warn("rate limit approaching")
	Warn(fields ...any)

	// Error logs an error-level message. The error is always required.
	// Pass optional key-value pairs after the error: Error(err, "key", val)
	Error(err error, fields ...any)

	// With returns a new Logger with the given key-value fields attached to every
	// subsequent log entry. Use to add context that applies across multiple calls:
	// log = log.With("requestId", id, "userId", uid)
	With(fields ...any) Logger
}
