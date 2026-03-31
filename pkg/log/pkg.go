package log

import (
	"context"
	"log/slog"
)

var ctxKey = &struct {
	name string
}{
	name: "log",
}

const (
	// LevelNotice is a custom slog level for notice messages.
	LevelNotice = slog.Level(10)
)

// FromContext returns the logger stored in the context, or the default logger if none is found.
func FromContext(ctx context.Context) *slog.Logger {
	v := ctx.Value(ctxKey)

	logger, ok := v.(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}

// NewContext returns a new context with the given logger attached.
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey, logger)
}
