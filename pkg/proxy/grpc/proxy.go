package grpc

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"http-benchmark/pkg/proxy"
	"log/slog"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
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
	FailTimeout time.Duration
}

type GRPCProxy struct {
	mu      sync.RWMutex
	id      string
	options *Options
	client  *grpc.ClientConn
	// target is set as a reverse proxy address
	target       string
	weight       uint
	failedCount  uint
	failExpireAt time.Time
}

func New(option Options) (*GRPCProxy, error) {

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(&rawCodec{})),
		grpc.WithIdleTimeout(30 * time.Minute),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(1024 * 1024 * 20)), // 20MB
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 20)), // 20MB
		grpc.WithDisableRetry(),
	}

	client, err := grpc.NewClient(option.Target, grpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("fail to dial backend: %v", err)
	}

	return &GRPCProxy{
		options: &option,
		client:  client,
	}, nil
}

func (p *GRPCProxy) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.options.MaxFails == 0 {
		return true
	}

	now := time.Now()
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

	now := time.Now()
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

func (p *GRPCProxy) Weight() uint {
	return p.weight
}

func (p *GRPCProxy) Target() string {
	return p.target
}

func (p *GRPCProxy) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(c, "proxy: grpc proxy panic recovered", slog.Any("error", r))
			ctx.Abort()
		}
	}()

	fullMethodName := string(ctx.Request.URI().Path())

	md := metadata.New(nil)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		md.Append(string(key), string(value))
	})
	grpcCtx := metadata.NewOutgoingContext(c, md)

	payload := ctx.Request.Body()
	if len(payload) < 5 {
		return
	}
	msgLen := binary.BigEndian.Uint32(payload[1:5])
	payload = payload[5 : 5+msgLen]

	var header, trailer metadata.MD

	var respBody []byte
	err := p.client.Invoke(grpcCtx, fullMethodName, payload, &respBody, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		slog.Error("fail to invoke grpc service", "error", err)
		handleGRPCError(ctx, err)
		return
	}

	if trailer.Len() == 0 {
		_ = ctx.Response.Header.Trailer().Set("grpc-status", "0")
	}

	// build http frame
	frame := make([]byte, len(respBody)+5)
	frame[0] = 0 // 0: not compression
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(respBody)))
	copy(frame[5:], respBody)

	for k, v := range header {
		for _, vv := range v {
			ctx.Response.Header.Add(k, vv)
		}
	}
	for k, v := range trailer {
		for _, vv := range v {
			_ = ctx.Response.Header.Trailer().Set(k, vv)
			ctx.Response.Header.Add("grpc-"+k, vv)
		}
	}

	ctx.SetStatusCode(200)
	ctx.Response.Header.Set("Content-Type", "application/grpc")
	ctx.Response.SetBody(frame)
}

func handleGRPCError(ctx *app.RequestContext, err error) {
	st, ok := status.FromError(err)
	if !ok {
		// If it's not a gRPC status error, create an internal error
		st = status.New(codes.Internal, err.Error())
	}

	// Set gspecific response headers
	code := fmt.Sprintf("%d", st.Code())
	_ = ctx.Response.Header.Trailer().Set("grpc-status", code)
	_ = ctx.Response.Header.Trailer().Set("grpc-message", st.Message())
	ctx.Response.Header.SetContentType("application/grpc")
	ctx.Response.Header.Set("grpc-status", code)
	ctx.Response.Header.Set("grpc-message", st.Message())

	// Include detailed information if available
	details := st.Proto().GetDetails()
	if len(details) > 0 {
		detailsBytes, err := proto.Marshal(st.Proto())
		if err == nil {
			ctx.Response.Header.Set("grpc-status-details-bin", base64.StdEncoding.EncodeToString(detailsBytes))
		}
	}

	// gRPC always uses 200 OK status code, actual status is in grpc-status header
	ctx.SetStatusCode(200)

	// Optionally write error information to the response body
	// Note: Some gRPC clients may not expect a response body in error cases
	errorFrame := makeGRPCErrorFrame(st)
	ctx.Response.SetBody(errorFrame)
}

func makeGRPCErrorFrame(st *status.Status) []byte {
	statusProto := st.Proto()
	serialized, _ := proto.Marshal(statusProto)
	frame := make([]byte, len(serialized)+5)
	frame[0] = 0 // 0: no compression
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(serialized)))
	copy(frame[5:], serialized)
	return frame
}
