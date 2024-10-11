package logging

import (
	"context"
	"fmt"
	"log/slog"

	slogcommon "github.com/samber/slog-common"
)

type LogStore interface {
	Insert(ctx context.Context, query string, values ...any)
	Close(ctx context.Context)
}

// Config contains the necessary to create persister.
type Config struct {
	Level                      slog.Leveler
	AttrFromContextExtractFunc []func(ctx context.Context) []slog.Attr
	LogStore                   LogStore
}

// Persister represents the log persister that will store logs in ScyllaDB.
type Persister struct {
	store       LogStore
	logLevel    slog.Leveler
	attrs       []slog.Attr
	groups      []string
	extractFunc []func(ctx context.Context) []slog.Attr
}

// NewPersister initializes a ScyllaDB session based on the provided config,
// and returns the Persister along with a session closer function.
func NewPersister(cfg *Config) *Persister {
	// Initialize the ScyllaDB cluster configuration

	if cfg.Level == nil {
		cfg.Level = slog.LevelDebug
	}

	if cfg.AttrFromContextExtractFunc == nil {
		cfg.AttrFromContextExtractFunc = []func(ctx context.Context) []slog.Attr{}
	}

	// Return the Persister and the closer function
	return &Persister{
		store:       cfg.LogStore,
		logLevel:    cfg.Level,
		attrs:       []slog.Attr{},
		groups:      []string{},
		extractFunc: cfg.AttrFromContextExtractFunc,
	}
}

func (p *Persister) Enabled(_ context.Context, level slog.Level) bool {
	return level >= p.logLevel.Level()
}

func (p *Persister) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Persister{
		logLevel: p.logLevel,
		store:    p.store,
		attrs:    slogcommon.AppendAttrsToGroup(p.groups, p.attrs, attrs...),
		groups:   p.groups,
	}
}

func (p *Persister) WithGroup(name string) slog.Handler {
	if name == "" {
		return p
	}

	return &Persister{
		logLevel: p.logLevel,
		store:    p.store,

		attrs:  p.attrs,
		groups: append(p.groups, name),
	}
}

// Handle implements the slog.Handler interface for log persistence.
func (p *Persister) Handle(ctx context.Context, record slog.Record) error {
	// Extract the log message and metadata
	var attrs []slog.Attr
	for _, fn := range p.extractFunc {
		attrs = append(attrs, fn(ctx)...)
	}

	output := converter(append(p.attrs, attrs...), p.groups, &record)
	recordAttrs := make(map[string]string)
	record.Attrs(func(a slog.Attr) bool {
		recordAttrs[a.Key] = fmt.Sprintf("%v", a.Value)
		return true
	})

	// Store the log message in ScyllaDB
	query := `INSERT INTO logs (timestamp, level, message, attributes) VALUES (?, ?, ?, ?)`
	p.store.Insert(
		ctx,
		query,
		output.Time,           // log timestamp
		output.Level.String(), // log level (info, error, etc.)
		output.Message,        // log message
		recordAttrs,           // log attributes as a map
	)

	return nil
}

func converter(
	loggerAttr []slog.Attr,
	groups []string,
	record *slog.Record,
) *slog.Record {
	// aggregate all attributes
	attrs := slogcommon.AppendRecordAttrsToAttrs(loggerAttr, groups, record)

	attrs = slogcommon.RemoveEmptyAttrs(attrs)

	output := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	output.AddAttrs(attrs...)
	return &output
}
