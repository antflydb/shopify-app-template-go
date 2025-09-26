package logging

import "context"

// Logger provides logic for using logs in code.
type Logger interface {
	// Named - returns a new logger with a chained name.
	Named(name string) Logger
	// Named - returns a new logger with a passed values in logging context.
	With(args ...any) Logger
	// WithContext - returns a new logger with a value from context.
	WithContext(ctx context.Context) Logger
	// Debug - logs in debug level.
	Debug(message string, args ...any)
	// Info - logs in info level.
	Info(message string, args ...any)
	// Warn - logs in warn level.
	Warn(message string, args ...any)
	// Error - logs in error level.
	Error(message string, args ...any)
	// Fatal - logs and exits program with status 1.
	Fatal(message string, args ...any)
}
