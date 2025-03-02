package gateway

import (
	"context"
	"time"
)

type resolver interface {
	Lookup(ctx context.Context, host string) ([]string, error)
	Valid() time.Duration
}
