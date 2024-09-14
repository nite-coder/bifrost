package proxy

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
)

var (
	ErrMaxFailedCount = errors.New("proxy: reach max failed count")
)

type Proxy interface {
	ID() string
	Target() string
	Weight() uint
	IsAvailable() bool
	AddFailedCount(count uint) error
	ServeHTTP(c context.Context, ctx *app.RequestContext)
}
