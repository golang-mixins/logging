// Package logrus represents implementations the interface logging.Log.
// The package is focused on the GELF Payload Specification logging format (http://docs.graylog.org/en/2.4/pages/gelf.html).
// Package implements adding to log the "timestamp" field, and gives the opportunity to specify fields:
// - "level";
// - "short_message".
// Adding marked fields is implemented in "github.com/sirupsen/logrus", the capabilities of which are implemented in the package.
package logrus

import (
	"context"
	"io"
	"os"
	"sync"

	"go.opencensus.io/trace"

	"github.com/golang-mixins/logging"
	log "github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

const (
	// DebugLevel - determines the level of logging "debug".
	DebugLevel string = "debug"
	// InfoLevel - determines the level of logging "info".
	InfoLevel string = "info"
	// WarnLevel - determines the level of logging "warning".
	WarnLevel string = "warning"
	// ErrorLevel - determines the level of logging "error".
	ErrorLevel string = "error"
	// FatalLevel - determines the level of logging "fatal".
	FatalLevel string = "fatal"
	// PanicLevel - determines the level of logging "panic".
	PanicLevel string = "panic"
)

type contextKey struct {
	name string
}

var ctxValue = &contextKey{"logger"}

// entry implements log.Entry.
type entry struct {
	*log.Entry
	breaker chan context.Context
}

// WithValues wraps the logging.Values in log.Values and returns an instance of the entry in the form of interface logging.Entry.
// Provides an instance of an entry with chaining implementation of fields.
func (e *entry) WithValues(v logging.Values) logging.Entry {
	return &entry{e.WithFields(log.Fields(v)), e.breaker}
}

// GetValues provides the current context of the instance.
func (e *entry) GetValues() logging.Values {
	return logging.Values(e.Data)
}

// GracefulFatal performs a soft fatal telling the fatal signal to the main application.
func (e *entry) GracefulFatal(ctx context.Context) {
	var span *trace.Span
	ctx, span = trace.StartSpan(ctx, "graceful fatal")
	defer span.End()

	go func() { defer func() { _ = recover() }(); e.breaker <- ctx }()
}

// FromContext returns the Entry stored in a context, or nil if there isn't one.
func (e *entry) FromContext(ctx context.Context) logging.Entry {
	logger, _ := ctx.Value(ctxValue).(*entry)
	if logger == nil {
		return nil
	}
	return logger
}

// NewContext returns the new context with entry.
func (e *entry) NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxValue, e)
}

// ContextLogger implements log.Log.
type ContextLogger struct {
	*log.Logger
	mutex   *sync.RWMutex
	breaker chan context.Context
}

// WithValues wraps the logging.Values in log.Values and returns an instance of the entry in the form of interface logging.Entry.
// Provides an instance of an entry with primary implementation of fields.
func (cl *ContextLogger) WithValues(v logging.Values) logging.Entry {
	return &entry{cl.WithFields(log.Fields(v)), cl.breaker}
}

// FromContext returns the Entry stored in a context, or nil if there isn't one.
func (cl *ContextLogger) FromContext(ctx context.Context) logging.Entry {
	e, _ := ctx.Value(ctxValue).(*entry)
	if e == nil {
		return nil
	}
	return e
}

// NewContext returns the new context with entry.
func (cl *ContextLogger) NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxValue, cl.WithFields(log.Fields(nil)))
}

// GracefulFatal performs a soft fatal telling the fatal signal to the main application.
func (cl *ContextLogger) GracefulFatal(ctx context.Context) {
	var span *trace.Span
	ctx, span = trace.StartSpan(ctx, "graceful fatal")
	defer span.End()

	go func() { defer func() { _ = recover() }(); cl.breaker <- ctx }()
}

// GetValues provides the current context of the instance.
func (cl *ContextLogger) GetValues() logging.Values {
	return logging.Values(log.Fields{})
}

// AddHooks adds hooks from the cut of the hooks in the argument. If the hook does not match the interface log.Hook, returns an error.
func (cl *ContextLogger) AddHooks(hooks ...interface{}) error {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	for _, v := range hooks {
		hook, ok := v.(log.Hook)
		if !ok || hook == nil {
			return xerrors.Errorf("value '%+v' is does not match the interface Hook", v)
		}
		cl.AddHook(hook)
	}
	return nil
}

// New is a ContextLogger constructor.
// New takes argument outputs. Outputs is an optional argument in the slice the outputs to the files of the additional log.
// - If outputs is empty, then only std output on /dev/stderr is used.
// - If outputs is not empty, then values of the slice is used to output the log to an additional files along with the std output.
func New(breaker chan context.Context, level string, outputs ...string) (logging.Logger, error) {
	if breaker == nil {
		return nil, xerrors.New("breaker can't be nil")
	}
	logger := log.New()
	logger.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "02.01.2006 15:04:05",
		FieldMap: log.FieldMap{
			log.FieldKeyFile:        "file",
			log.FieldKeyFunc:        "func",
			log.FieldKeyLogrusError: "logger_error",
			log.FieldKeyTime:        "timestamp",
			log.FieldKeyLevel:       "level",
			log.FieldKeyMsg:         "message",
		},
	},
	)
	writers := append(make([]io.Writer, 0, len(outputs)+1), os.Stdout)
	for _, v := range outputs {
		file, err := os.OpenFile(v, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, xerrors.Errorf("error open file path '%s': %w", v, err)
		}
		writers = append(writers, file)
	}
	logger.Out = io.MultiWriter(writers...)
	logger.SetReportCaller(true)
	lvl, err := log.ParseLevel(level)
	if err != nil {
		return nil, xerrors.Errorf("error parse level value '%s': %w", level, err)
	}
	logger.SetLevel(lvl)
	return &ContextLogger{
		logger,
		&sync.RWMutex{},
		breaker,
	}, nil
}
