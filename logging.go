// Package logging presents interface (and its implementation sets) of a structured logger for Go.
// The interface is focused on the GELF Payload Specification logging format (http://docs.graylog.org/en/2.4/pages/gelf.html),
// but provides support for the universal use of logging.
// Recommendations for the implementation of GELF Payload Specification field formatting:
// - "version" - adds the sender of the message to the log.
// - "host" - adds the used hook (for example, GrayLog Hook).
// - "short_message" - adds the sender of the message to the log.
// - "full_message" - adds the sender of the message to the log.
// - "timestamp" - adds logging.
// - "level" - adds the sender of the message to the log.
// - "facility" - adds the sender of the message to the log.
// - "line" - adds a used hook (for example, GrayLog Hook).
// - "file" - adds a used hook (for example, GrayLog Hook).
// - additional fields - adds the sender of the message to the log.
package logging

import (
	"context"
	"io"
)

// Values built in type for processing fields in context.
type Values map[string]interface{}

// Entry provides recording to logging.
type Entry interface {
	// Debug captures a logging entry with a "debug" level.
	Debug(args ...interface{})
	// Info captures a logging entry with a "info" level.
	Info(args ...interface{})
	// Warning captures a logging entry with a "warning" level.
	Warning(args ...interface{})
	// Error captures a logging entry with a "error" level.
	Error(args ...interface{})
	// Fatal captures a logging entry with a "fatal" level.
	Fatal(args ...interface{})
	// Panic captures a logging entry with a "panic" level.
	Panic(args ...interface{})
	// GracefulFatal elegantly completes the system, reporting the main process of the system.
	GracefulFatal(ctx context.Context)
	// Writer returns *io.PipeWriter.
	Writer() *io.PipeWriter
	// WithValues enriches Entry Values.
	WithValues(v Values) Entry
	// GetValues returns Entry Values.
	GetValues() Values
	// FromContext returns the Entry stored in a context, or nil if there isn't one.
	FromContext(ctx context.Context) Entry
	// NewContext returns the new context with Entry.
	NewContext(ctx context.Context) context.Context
	// TruncateToMaxValueLength returns a value optimized for the maximum supported length.
	TruncateToMaxValueLength(value []byte) []byte
}

// Logger provides logging functionality.
type Logger interface {
	Entry
	// AddHooks adds hooks to the Logger.
	AddHooks(hooks ...interface{}) error
}
