package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
)

type FanOut struct {
	handlers []slog.Handler
}

// Fanout distributes records to multiple slog.Handler in parallel
func NewFanOut(handlers ...slog.Handler) slog.Handler {
	return &FanOut{
		handlers: handlers,
	}
}

func (f FanOut) Enabled(ctx context.Context, level slog.Level) bool {
	for i := range f.handlers {
		if f.handlers[i].Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (f FanOut) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for i := range f.handlers {
		if f.handlers[i].Enabled(ctx, record.Level) {
			err := try(func() error {
				return f.handlers[i].Handle(ctx, record.Clone())
			})
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	// If errs is empty, or contains only nil errors, this returns nil
	return errors.Join(errs...)
}

func (f FanOut) WithAttrs(attrs []slog.Attr) slog.Handler {
	for _, handler := range f.handlers {
		handler = handler.WithAttrs(slices.Clone(attrs))
	}

	return f
}

func (f FanOut) WithGroup(name string) slog.Handler {
	if name == "" {
		return f
	}

	for _, handler := range f.handlers {
		handler = handler.WithGroup(name)
	}

	return f
}

func try(callback func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("unexpected error: %+v", r)
			}
		}
	}()

	err = callback()

	return
}
