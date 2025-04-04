package mirror

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func init() {
	_ = middleware.RegisterMiddleware("mirror", func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("mirror middleware params is empty or invalid")
		}

		opts := &Options{}

		err := mapstructure.Decode(params, &opts)
		if err != nil {
			return nil, fmt.Errorf("mirror middleware params is invalid: %w", err)
		}

		if opts.ServiceID == "" {
			return nil, errors.New("mirror: service_id can't be empty")
		}

		m := NewMiddleware(*opts)

		return m.ServeHTTP, nil
	})
}

type Options struct {
	ServiceID string `mapstructure:"service_id"`
	QueueSize int64  `mapstructure:"queue_size"`
}

type MirrorMiddleware struct {
	options *Options
	queue   chan *mirrorContext
}

type mirrorContext struct {
	logger *slog.Logger
	hzCtx  *app.RequestContext
}

func NewMiddleware(options Options) *MirrorMiddleware {
	if options.QueueSize <= 0 {
		options.QueueSize = 10000
	}

	m := &MirrorMiddleware{
		options: &options,
		queue:   make(chan *mirrorContext, options.QueueSize),
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				stackTrace := runtime.StackTrace()
				logger.Error("mirror panic recovered", slog.Any("panic", r), slog.String("stack", stackTrace))
			}
		}()
		m.Run()
	}()

	return m
}

func (m *MirrorMiddleware) Run() {
	for mctx := range m.queue {
		bifrost := gateway.GetBifrost()

		if bifrost == nil {
			continue
		}

		if !bifrost.IsActive() {
			break
		}

		svc, found := bifrost.Service(m.options.ServiceID)

		if !found {
			slog.Warn("mirror: service not found", "service_id", m.options.ServiceID)
			continue
		}

		middlewares := svc.Middlewares()
		middlewares = append(middlewares, svc.ServeHTTP)

		ctx := log.NewContext(context.Background(), mctx.logger)

		c := mctx.hzCtx
		c.SetIndex(-1)
		c.SetHandlers(middlewares)
		c.Next(ctx)
	}
}

func (m *MirrorMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	mctx := &mirrorContext{
		logger: log.FromContext(ctx),
		hzCtx:  c.Copy(),
	}

	select {
	case m.queue <- mctx:
	default:
	}

	c.Next(ctx)
}
