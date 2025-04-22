package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/network"
	"github.com/cloudwego/hertz/pkg/protocol"
	reqHelper "github.com/cloudwego/hertz/pkg/protocol/http1/req"
	respHelper "github.com/cloudwego/hertz/pkg/protocol/http1/resp"
	"github.com/nite-coder/bifrost/internal/pkg/task"
	"golang.org/x/net/http/httpguts"
)

func upgradeReqType(h *protocol.RequestHeader) string {
	if !httpguts.HeaderValuesContainsToken(h.GetAll("Connection"), "Upgrade") {
		return ""
	}
	return h.Get("Upgrade")
}

func upgradeRespType(h *protocol.ResponseHeader) string {
	if !httpguts.HeaderValuesContainsToken(h.GetAll("Connection"), "Upgrade") {
		return ""
	}
	return h.Get("Upgrade")
}

func (p *HTTPProxy) roundTrip(ctx context.Context, clientCtx *app.RequestContext, req *protocol.Request, resp *protocol.Response) error {
	dailer := p.client.GetOptions().Dialer

	host := string(req.Host())

	if bytes.EqualFold(req.Scheme(), https) {
		host = utils.AddMissingPort(host, true)
	} else {
		host = utils.AddMissingPort(host, false)
	}

	backendConn, err := dailer.DialConnection("tcp", host, p.client.GetOptions().DialTimeout, p.client.GetOptions().TLSConfig)
	if err != nil {
		return err
	}

	err = reqHelper.Write(req, backendConn)
	if err != nil {
		return err
	}

	err = backendConn.Flush()
	if err != nil {
		return err
	}

	backendHeader := &resp.Header
	backendHeader.SetNoDefaultContentType(true)
	err = respHelper.ReadHeader(backendHeader, backendConn)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusSwitchingProtocols {
		err := fmt.Errorf("backend returns status is not 101, status code: %d", resp.StatusCode())
		p.handleError(ctx, clientCtx, err)
		return err
	}

	reqUpType := upgradeReqType(&req.Header)
	resUpType := upgradeRespType(backendHeader)

	if !IsASCIIPrint(resUpType) { // We know reqUpType is ASCII, it's checked by the caller.
		p.handleError(ctx, clientCtx, err)
		return err
	}
	if !strings.EqualFold(reqUpType, resUpType) {
		err := fmt.Errorf("backend tried to switch protocol %q when %q was requested", resUpType, reqUpType)
		p.handleError(ctx, clientCtx, err)
		return err
	}

	clientConn := clientCtx.GetConn()

	_, err = clientConn.Write(backendHeader.Header())
	if err != nil {
		p.handleError(ctx, clientCtx, fmt.Errorf("write header to client error %w", err))
		return err
	}

	err = clientConn.Flush()
	if err != nil {
		p.handleError(ctx, clientCtx, fmt.Errorf("flush header to client error %w", err))
		return err
	}

	p.handleUpgradeResponse(ctx, clientConn, backendConn)
	return nil
}

func (p *HTTPProxy) handleUpgradeResponse(ctx context.Context, clientConn network.Conn, backendConn network.Conn) {
	backConnCloseCh := make(chan bool)
	go func() {
		task.Runner(ctx, func() {
			// Ensure that the cancellation of a request closes the backend.
			// See issue https://golang.org/issue/35559.
			select {
			case <-ctx.Done():
			case <-backConnCloseCh:
			}
			_ = backendConn.Close()
		})
	}()
	defer close(backConnCloseCh)

	errc := make(chan error, 1)
	spc := switchProtocolCopier{user: clientConn, backend: backendConn}

	go spc.copyToBackend(errc)
	go spc.copyFromBackend(errc)

	erra := <-errc

	if erra != nil {
		slog.ErrorContext(ctx, "copyToBackend:", "error", erra)
		return
	}

	slog.Info("finish proxy upgrade")
}

// switchProtocolCopier exists so goroutines proxying data back and
// forth have nice names in stacks.
type switchProtocolCopier struct {
	user, backend io.ReadWriter
}

func (c switchProtocolCopier) copyFromBackend(errc chan<- error) {
	_, err := io.Copy(c.user, c.backend)
	errc <- err
}

func (c switchProtocolCopier) copyToBackend(errc chan<- error) {
	_, err := io.Copy(c.backend, c.user)
	errc <- err
}
