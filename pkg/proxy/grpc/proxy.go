package grpc

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/url"
	"sync"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type Options struct {
	Target      string
	TLSVerify   bool
	Weight      uint
	MaxFails    uint
	Timeout     time.Duration
	FailTimeout time.Duration
	DailOptions []grpc.DialOption
}

type GRPCProxy struct {
	mu      sync.RWMutex
	id      string
	options *Options
	client  grpc.ClientConnInterface
	tracer  trace.Tracer
	// target is set as a reverse proxy address
	target       string
	targetHost   string
	weight       uint32
	failedCount  uint
	failExpireAt time.Time
}

func New(options Options) (*GRPCProxy, error) {
	addr, err := url.Parse(options.Target)
	if err != nil {
		return nil, fmt.Errorf("proxy: grpc proxy fail to parse target url; %w", err)
	}

	if options.Weight == 0 {
		options.Weight = 1
	}

	grpcOptions := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.ForceCodec(&rawCodec{})),
		grpc.WithDisableRetry(),
	}

	if !options.TLSVerify {
		grpcOptions = append(grpcOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if len(options.DailOptions) > 0 {
		grpcOptions = append(grpcOptions, options.DailOptions...)
	}

	client, err := grpc.NewClient(addr.Host, grpcOptions...)
	if err != nil {
		return nil, fmt.Errorf("fail to dial backend: %w", err)
	}

	return &GRPCProxy{
		id:         uuid.New().String(),
		targetHost: addr.Host,
		options:    &options,
		client:     client,
		tracer:     otel.Tracer("bifrost"),
	}, nil
}

func (p *GRPCProxy) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.options.MaxFails == 0 {
		return true
	}

	now := timecache.Now()
	if now.After(p.failExpireAt) {
		return true
	}

	if p.failedCount < p.options.MaxFails {
		return true
	}

	return false
}

func (p *GRPCProxy) AddFailedCount(count uint) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := timecache.Now()
	if now.After(p.failExpireAt) {
		p.failExpireAt = now.Add(p.options.FailTimeout)
		p.failedCount = count
	} else {
		p.failedCount += count
	}

	if p.options.MaxFails > 0 && p.failedCount >= p.options.MaxFails {
		return proxy.ErrMaxFailedCount
	}

	return nil
}

// ID return proxy's ID
func (p *GRPCProxy) ID() string {
	return p.id
}

func (p *GRPCProxy) Weight() uint32 {
	return p.weight
}

func (p *GRPCProxy) Target() string {
	return p.target
}

func (p *GRPCProxy) Close() error {
	if p.client != nil {
		if closer, ok := p.client.(io.Closer); ok {
			err := closer.Close()
			if err != nil {
				return err
			}
		}
	}

	p = nil
	return nil
}

// ServeHTTP implements the http.Handler interface
func (p *GRPCProxy) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)

	defer func() {
		if r := recover(); r != nil {
			stackTrace := runtime.StackTrace()
			logger.ErrorContext(ctx, "proxy: grpc proxy panic recovered", slog.Any("error", r), slog.String("stack", stackTrace))
			c.Abort()
		}
	}()

	if p.client == nil {
		logger.ErrorContext(ctx, "proxy: grpc proxy client is nil", slog.Any("error", "grpc proxy client is nil"))
		c.Abort()
		return
	}

	// Get the full method name from the request URL
	fullMethodName := string(c.Request.URI().Path())

	// Set the upstream address
	c.Set(variable.UpstreamRequestHost, p.targetHost)

	// Create a new metadata object
	md := metadata.New(nil)

	// Iterate over the request headers and add them to the metadata
	c.Request.Header.VisitAll(func(key, value []byte) {
		md.Append(string(key), string(value))
	})

	// Create a new grpc context with the metadata
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Check if the request payload is valid
	payload := c.Request.Body()
	if len(payload) < 5 {
		logger.ErrorContext(ctx, "proxy: grpc proxy request payload is invalid", slog.Any("error", "grpc proxy request payload is invalid"))
		return
	}

	// Get the length of the message
	msgLen := binary.BigEndian.Uint32(payload[1:5])

	// Get the message payload
	payload = payload[5 : 5+msgLen]

	// Create a new header and trailer metadata
	var header, trailer metadata.MD

	// Create a new response body
	var respBody []byte

	// If a timeout is set, create a new context with the timeout
	if p.options.Timeout > 0 {
		ctxTimeout, cancel := context.WithTimeout(ctx, p.options.Timeout)
		ctx = ctxTimeout
		defer cancel()
	}

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
	}

	ctx, span := p.tracer.Start(ctx, fullMethodName, spanOptions...)

	labels := []attribute.KeyValue{
		attribute.String("http.method", "POST"),
		attribute.String("http.scheme", string(c.Request.Scheme())),
		attribute.String("http.path", fullMethodName),
		attribute.String("http.host", p.targetHost),
		attribute.String("protocol", "grpc"),
	}
	span.SetAttributes(labels...)

	// Call the grpc client with the request
	err := p.client.Invoke(ctx, fullMethodName, payload, &respBody, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		// Handle the error
		p.handleGRPCError(ctx, c, err)
		return
	}

	span.SetStatus(otelcodes.Ok, "")
	span.End()

	// If there is no trailer, set the grpc-status header to 0
	if trailer.Len() == 0 {
		_ = c.Response.Header.Trailer().Set("grpc-status", "0")
	}

	// Build the http frame
	frame := make([]byte, len(respBody)+5)

	// Set the first byte to 0, indicating no compression
	frame[0] = 0

	// Set the length of the message
	val := len(respBody)
	if val > math.MaxUint32 || val < 0 {
		logger.Error("proxy: grpc proxy response payload is overflow")
		err := errors.New("proxy: grpc proxy response payload is overflow")
		_ = c.Error(err)
		return
	}

	binary.BigEndian.PutUint32(frame[1:5], uint32(val))

	// Copy the response body to the frame
	copy(frame[5:], respBody)

	// Iterate over the header and trailer metadata and add them to the response headers
	for k, v := range header {
		for _, vv := range v {
			c.Response.Header.Add(k, vv)
		}
	}

	for k, v := range trailer {
		for _, vv := range v {
			_ = c.Response.Header.Trailer().Set(k, vv)
			c.Response.Header.Add("grpc-"+k, vv)
		}
	}

	// Set the status code and content type
	c.SetStatusCode(200)
	c.Response.Header.Set("Content-Type", "application/grpc")
	c.Response.SetBody(frame)
}

func (p *GRPCProxy) handleGRPCError(ctx context.Context, c *app.RequestContext, err error) {
	if err == nil {
		return
	}

	logger := log.FromContext(ctx)

	val, _ := variable.Get(variable.HTTPRequestPath, c)
	originalPath, _ := cast.ToString(val)

	st, ok := status.FromError(err)
	if !ok {
		// If it's not a gRPC status error, create an internal error
		st = status.New(codes.Internal, err.Error())
	}

	c.Set(variable.GRPCStatusCode, uint32(st.Code()))

	logger.Error("fail to invoke grpc server",
		slog.String("error", err.Error()),
		slog.String("original_path", originalPath),
		slog.String("upstream", p.targetHost+string(c.Request.Path())),
		slog.String("grpc_status", st.Code().String()),
		slog.String("grpc_message", st.Message()),
	)

	// Set gspecific response headers
	code := fmt.Sprintf("%d", st.Code())
	_ = c.Response.Header.Trailer().Set("grpc-status", code)
	_ = c.Response.Header.Trailer().Set("grpc-message", st.Message())
	c.Response.Header.SetContentType("application/grpc")
	c.Response.Header.Set("grpc-status", code)
	c.Response.Header.Set("grpc-message", st.Message())

	span := trace.SpanFromContext(ctx)
	span.SetStatus(otelcodes.Error, st.Message())
	labels := []attribute.KeyValue{
		attribute.String("grpc.status", st.Code().String()),
		attribute.String("grpc.message", st.Message()),
		attribute.String("original_path", originalPath),
		attribute.String("error", err.Error()),
	}
	span.SetAttributes(labels...)
	span.End()

	switch st.Code() {
	case codes.Unavailable, codes.Unknown, codes.Unimplemented:
		err := p.AddFailedCount(1)
		if err != nil {
			logger.Warn("upstream server temporarily disabled")
		}
	default:
	}

	// Include detailed information if available
	details := st.Proto().GetDetails()
	if len(details) > 0 {
		detailsBytes, err := proto.Marshal(st.Proto())
		if err == nil {
			c.Response.Header.Set("grpc-status-details-bin", base64.StdEncoding.EncodeToString(detailsBytes))
		}
	}

	// gRPC always uses 200 OK status code, actual status is in grpc-status header
	c.SetStatusCode(200)

	// Optionally write error information to the response body
	// Note: Some gRPC clients may not expect a response body in error cases
	errorFrame := makeGRPCErrorFrame(ctx, st)
	c.Response.SetBody(errorFrame)
}

func makeGRPCErrorFrame(ctx context.Context, st *status.Status) []byte {
	statusProto := st.Proto()
	serialized, _ := proto.Marshal(statusProto)

	val := len(serialized)
	if val > math.MaxUint32-5 {
		// Check for potential overflow
		// Handle the error appropriately, e.g., log and return an empty frame
		logger := log.FromContext(ctx)
		logger.Error("proxy: Serialized data too large to create gRPC error frame")
		return []byte{}
	}

	frame := make([]byte, val+5)
	frame[0] = 0 // 0: no compression
	binary.BigEndian.PutUint32(frame[1:5], uint32(val))
	copy(frame[5:], serialized)
	return frame
}
