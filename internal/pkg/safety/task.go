package safety

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/nite-coder/blackbear/pkg/cast"
)

var (
	Go func(ctx context.Context, f func())
)

func init() {
	Go = func(ctx context.Context, f func()) {
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch v := r.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("%v", v)
				}
				stackTrace := debug.Stack()
				slog.Error("safty Go panic recovered",
					slog.String("error", err.Error()),
					slog.String("stack", cast.B2S(stackTrace)),
				)
			}
		}()
		f()
	}
}
