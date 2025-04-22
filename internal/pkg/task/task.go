package task

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nite-coder/bifrost/internal/pkg/runtime"
)

var (
	Runner func(ctx context.Context, f func())
)

func init() {
	Runner = func(ctx context.Context, f func()) {
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch v := r.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("%v", v)
				}
				stackTrace := runtime.StackTrace()
				slog.Error("runTask panic recovered",
					slog.String("error", err.Error()),
					slog.String("stack", stackTrace),
				)
			}
		}()
		f()
	}
}
