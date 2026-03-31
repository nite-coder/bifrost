package safety

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/nite-coder/blackbear/pkg/cast"
)

// Go executes the given function with panic recovery.
// It is synchronous; it does NOT spawn a goroutine.
// Use 'go Go(ctx, f)' to spawn a safe goroutine.
var Go func(ctx context.Context, f func())

func init() {
	// Default implementation runs the function synchronously with panic recovery.
	Go = func(_ context.Context, f func()) {
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
				slog.Error("safety Go panic recovered",
					slog.String("error", err.Error()),
					slog.String("stack", cast.B2S(stackTrace)),
				)
			}
		}()
		f()
	}
}
