package log

import (
	"context"
	"log/slog"
)

var (
	ctxKey = &struct {
		name string
	}{
		name: "log",
	}
)

const (
	LevelNotice = slog.Level(10)
)

func FromContext(ctx context.Context) *slog.Logger {
	v := ctx.Value(ctxKey)

	logger, ok := v.(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}

func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey, logger)
}
