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
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"github.com/nite-coder/blackbear/pkg/cast"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/tracing"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const grpcHeaderLen = 5

// Options defines the configuration for a GRPC proxy instance.
type Options struct {
	Target           string
	DailOptions      []grpc.DialOption
	Timeout          time.Duration
	TLSVerify        bool
	IsTracingEnabled bool
	ServiceID        string
	Endpoint         *proxy.Endpoint
}

// Proxy implements a reverse proxy for gRPC services.
type Proxy struct {
	client     grpc.ClientConnInterface
	options    *Options
	id         string
	target     string
	targetHost string
	endpoint   atomic.Pointer[proxy.Endpoint]
}

// New creates a new GRPCProxy instance with the given options.
func New(options Options) (*Proxy, error) {
	addr, err := url.Parse(options.Target)
	if err != nil {
		return nil, fmt.Errorf("proxy: gRPC proxy failed to parse target URL: %w", err)
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
		return nil, fmt.Errorf("failed to dial backend: %w", err)
	}

	if options.Endpoint == nil {
		return nil, errors.New("endpoint cannot be nil")
	}
	endpoint := options.Endpoint

	r := &Proxy{
		id:         uuid.New().String(),
		target:     options.Target,
		targetHost: addr.Host,
		options:    &options,
		client:     client,
	}
	r.endpoint.Store(endpoint)
	return r, nil
}

// Endpoint returns the endpoint info associated with this proxy.
func (p *Proxy) Endpoint() *proxy.Endpoint {
	return p.endpoint.Load()
}

// SetEndpoint updates the endpoint info associated with this proxy.
func (p *Proxy) SetEndpoint(ep *proxy.Endpoint) {
	p.endpoint.Store(ep)
}

// ID return proxy's ID.
func (p *Proxy) ID() string {
	return p.id
}

// Target returns the target URL of the upstream gRPC server.
func (p *Proxy) Target() string {
	return p.target
}

// Close closes the underlying gRPC client connection.
func (p *Proxy) Close() error {
	if p.client != nil {
		if closer, ok := p.client.(io.Closer); ok {
			err := closer.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ServeHTTP implements the http.Handler interface.
func (p *Proxy) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)
	defer func() {
		if r := recover(); r != nil {
			stackTrace := cast.B2S(debug.Stack())
			logger.ErrorContext(
				ctx,
				"proxy: gRPC proxy panic recovered",
				slog.Any("error", r),
				slog.String("stack", stackTrace),
			)
			c.Abort()
		}
	}()
	if p.client == nil {
		logger.ErrorContext(ctx, "proxy: gRPC proxy client is nil", slog.Any("error", "gRPC proxy client is nil"))
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
	if len(payload) < grpcHeaderLen {
		logger.WarnContext(
			ctx,
			"proxy: gRPC proxy request payload is invalid",
			slog.Any("error", "gRPC proxy request payload is invalid"),
		)
		return
	}
	// Get the length of the message
	msgLen := binary.BigEndian.Uint32(payload[1:grpcHeaderLen])
	// Check if the payload is large enough
	if uint64(len(payload)) < grpcHeaderLen+uint64(msgLen) {
		logger.WarnContext(ctx, "proxy: gRPC proxy request payload length mismatch",
			slog.Any("declared_len", msgLen),
			slog.Any("actual_len", len(payload)-grpcHeaderLen),
		)
		c.SetStatusCode(http.StatusBadRequest) // Bad Request
		return
	}
	// Get the message payload
	payload = payload[grpcHeaderLen : grpcHeaderLen+msgLen]
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

	if p.options.IsTracingEnabled {
		tracer := otel.Tracer("bifrost")
		if tracer != nil {
			var span trace.Span
			spanOptions := []trace.SpanStartOption{
				trace.WithSpanKind(trace.SpanKindClient),
			}
			ctx, span = tracer.Start(ctx, fullMethodName, spanOptions...)
			tracing.InjectGRPCMetadata(ctx, md)
			defer func() {
				// Extract service and method names.
				// Format: /<service-name>/<method-name>
				serviceName := ""
				methodName := ""
				if idx := strings.LastIndex(fullMethodName, "/"); idx != -1 {
					methodName = fullMethodName[idx+1:]
					serviceName = fullMethodName[1:idx]
				}

				labels := []attribute.KeyValue{
					semconv.RPCSystemGRPC,
					semconv.RPCService(serviceName),
					semconv.RPCMethod(methodName),
					semconv.ServerAddress(p.targetHost),
				}

				grpcStatusCode, found := c.Get(variable.GRPCStatusCode)
				if found {
					code, ok := grpcStatusCode.(codes.Code)
					if ok {
						if code != codes.OK {
							span.SetStatus(otelcodes.Error, c.GetString(variable.GRPCMessage))
							labels = append(labels, attribute.Int64("rpc.grpc.status_code", int64(code)))
							labels = append(
								labels,
								attribute.String("rpc.grpc.message", c.GetString(variable.GRPCMessage)),
							)
						} else {
							span.SetStatus(otelcodes.Ok, "")
						}
					}
				}

				span.SetAttributes(labels...)
				span.End()
			}()
		}
	}

	// Call the grpc client with the request
	err := p.client.Invoke(ctx, fullMethodName, payload, &respBody, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		// Handle the error
		p.handleGRPCError(ctx, c, err)
		return
	}

	c.Set(variable.GRPCStatusCode, codes.OK)

	// Build the http frame
	val := len(respBody)
	if val > math.MaxInt-grpcHeaderLen || val > math.MaxUint32-grpcHeaderLen {
		logger.Error("proxy: gRPC proxy response payload overflow")
		err := errors.New("proxy: gRPC proxy response payload overflow")
		_ = c.Error(err)
		return
	}

	frame := make([]byte, val+grpcHeaderLen)
	// Set the first byte to 0, indicating no compression
	frame[0] = 0
	// Set the length of the message
	binary.BigEndian.PutUint32(frame[1:grpcHeaderLen], uint32(val))
	// Copy the response body to the frame
	copy(frame[grpcHeaderLen:], respBody)

	// Iterate over the header and trailer metadata and add them to the response headers
	for k, v := range header {
		for _, vv := range v {
			c.Response.Header.Add(k, vv)
		}
	}

	// Always ensure grpc-status is present in trailers
	_ = c.Response.Header.Trailer().Set("grpc-status", "0")
	for k, v := range trailer {
		for _, vv := range v {
			_ = c.Response.Header.Trailer().Set(k, vv)
			c.Response.Header.Add("grpc-"+k, vv)
		}
	}
	// Set the status code and content type
	c.SetStatusCode(http.StatusOK)
	c.Response.Header.Set("Content-Type", "application/grpc")
	c.Response.SetBody(frame)
}

func (p *Proxy) handleGRPCError(ctx context.Context, c *app.RequestContext, err error) {
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
	c.Set(variable.GRPCStatusCode, st.Code())
	c.Set(variable.GRPCMessage, st.Message())
	logger.Error("failed to invoke gRPC server",
		slog.String("error", err.Error()),
		slog.String("original_path", originalPath),
		slog.String("upstream", p.targetHost+string(c.Request.Path())),
		slog.String("grpc_status", st.Code().String()),
		slog.String("grpc_message", st.Message()),
	)
	// Set gspecific response headers
	code := strconv.Itoa(int(st.Code()))
	_ = c.Response.Header.Trailer().Set("grpc-status", code)
	_ = c.Response.Header.Trailer().Set("grpc-message", st.Message())
	c.Response.Header.SetContentType("application/grpc")
	switch st.Code() {
	case codes.Unavailable, codes.Unknown, codes.Unimplemented, codes.Internal:
		ep := p.Endpoint()
		if ep != nil && ep.HealthState != nil {
			ep.HealthState.RecordFailure()
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
	c.SetStatusCode(http.StatusOK)
	// Optionally write error information to the response body
	// Note: Some gRPC clients may not expect a response body in error cases
	errorFrame := makeGRPCErrorFrame(ctx, st)
	c.Response.SetBody(errorFrame)
}

func makeGRPCErrorFrame(ctx context.Context, st *status.Status) []byte {
	statusProto := st.Proto()
	serialized, _ := proto.Marshal(statusProto)
	payloadLen := len(serialized)
	payloadLen64 := uint64(payloadLen)
	// Ensure payload length fits within the 32-bit gRPC length field.
	if payloadLen64 > math.MaxUint32 {
		logger := log.FromContext(ctx)
		logger.Error("proxy: serialized data too large to create gRPC error frame")
		return []byte{}
	}
	// Ensure total frame length (header + payload) fits within Go's maximum int.
	totalLen64 := payloadLen64 + uint64(grpcHeaderLen)
	if totalLen64 > math.MaxInt {
		logger := log.FromContext(ctx)
		logger.Error("proxy: total frame size too large to create gRPC error frame")
		return []byte{}
	}
	totalLen := int(totalLen64)
	frame := make([]byte, totalLen)
	frame[0] = 0 // 0: no compression
	payloadLen32 := uint32(payloadLen64)
	binary.BigEndian.PutUint32(frame[1:grpcHeaderLen], payloadLen32)
	copy(frame[grpcHeaderLen:], serialized)
	return frame
}
