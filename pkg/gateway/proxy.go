/*
 *	Copyright 2023 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2011 The Go Authors. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 *
 * This file may have been modified by CloudWeGo Authors. All CloudWeGo
 * Modifications are Copyright 2023 CloudWeGo Authors.
 */

package gateway

import (
	"bytes"
	"context"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"log/slog"
	"net"
	"net/textproto"
	"net/url"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	hertztracing "github.com/hertz-contrib/obs-opentelemetry/tracing"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/valyala/bytebufferpool"
)

type Proxy struct {
	client *client.Client

	// target is set as a reverse proxy address
	target string

	// transferTrailer is whether to forward Trailer-related header
	transferTrailer bool

	// saveOriginResponse is whether to save the original response header
	saveOriginResHeader bool

	// director must be a function which modifies the request
	// into a new request. Its response is then redirected
	// back to the original client unmodified.
	// director must not access the provided Request
	// after returning.
	director func(*protocol.Request)

	// modifyResponse is an optional function that modifies the
	// Response from the backend. It is called if the backend
	// returns a response at all, with any HTTP status code.
	// If the backend is unreachable, the optional errorHandler is
	// called without any call to modifyResponse.
	//
	// If modifyResponse returns an error, errorHandler is called
	// with its error value. If errorHandler is nil, its default
	// implementation is used.
	modifyResponse func(*protocol.Response) error

	// errorHandler is an optional function that handles errors
	// reaching the backend or errors from modifyResponse.
	//
	// If nil, the default is to log the provided error and return
	// a 502 Status Bad Gateway response.
	errorHandler func(*app.RequestContext, error)

	targetHost string

	weight int
}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

// newProxy returns a new ReverseProxy that routes
// URLs to the scheme, host, and base path provided in target. If the
// target's path is "/base" and the incoming request was for "/dir",
// the target request will be for /base/dir.
// newProxy does not rewrite the Host header.
// To rewrite Host headers, use ReverseProxy directly with a custom
// director policy.
//
// Note: if no config.ClientOption is passed it will use the default global client.Client instance.
// When passing config.ClientOption it will initialize a local client.Client instance.
// Using ReverseProxy.SetClient if there is need for shared customized client.Client instance.
func newProxy(target string, tracingEnabled bool, weight int, options ...hzconfig.ClientOption) (*Proxy, error) {
	addr, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	r := &Proxy{
		target:     target,
		targetHost: addr.Host,
		weight:     weight,
		director: func(req *protocol.Request) {
			req.Header.SetProtocol("HTTP/1.1")

			switch addr.Scheme {
			case "http":
				req.SetIsTLS(false)
			case "https":
				req.SetIsTLS(true)
			}

			req.SetRequestURI(cast.B2S(JoinURLPath(req, target)))
			//req.Header.SetHostBytes(req.URI().Host())
		},
	}

	if len(options) != 0 {
		c, err := client.NewClient(options...)
		if tracingEnabled {
			c.Use(hertztracing.ClientMiddleware())
		}
		if err != nil {
			return nil, err
		}
		r.client = c
	}
	return r, nil
}

func JoinURLPath(req *protocol.Request, target string) (path []byte) {
	aslash := req.URI().Path()[0] == '/'
	var bslash bool
	if strings.HasPrefix(target, "http") {
		// absolute path
		bslash = strings.HasSuffix(target, "/")
	} else {
		// default redirect to local
		bslash = strings.HasPrefix(target, "/")
		if bslash {
			target = fmt.Sprintf("%s%s", req.Host(), target)
		} else {
			target = fmt.Sprintf("%s/%s", req.Host(), target)
		}
		bslash = strings.HasSuffix(target, "/")
	}

	targetQuery := strings.Split(target, "?")
	buffer := bytebufferpool.Get()
	defer bytebufferpool.Put(buffer)

	_, _ = buffer.WriteString(targetQuery[0])
	switch {
	case aslash && bslash:
		_, _ = buffer.Write(req.URI().Path()[1:])
	case !aslash && !bslash:
		_, _ = buffer.Write([]byte{'/'})
		_, _ = buffer.Write(req.URI().Path())
	default:
		_, _ = buffer.Write(req.URI().Path())
	}
	if len(targetQuery) > 1 {
		_, _ = buffer.Write([]byte{'?'})
		_, _ = buffer.WriteString(targetQuery[1])
	}
	if len(req.QueryString()) > 0 {
		if len(targetQuery) == 1 {
			_, _ = buffer.Write([]byte{'?'})
		} else {
			_, _ = buffer.Write([]byte{'&'})
		}
		_, _ = buffer.Write(req.QueryString())
	}
	return buffer.Bytes()
}

// removeRequestConnHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func removeRequestConnHeaders(c *app.RequestContext) {
	c.Request.Header.VisitAll(func(k, v []byte) {
		if cast.B2S(k) == "Connection" {
			for _, sf := range strings.Split(cast.B2S(v), ",") {
				if sf = textproto.TrimString(sf); sf != "" {
					c.Request.Header.DelBytes(cast.S2B(sf))
				}
			}
		}
	})
}

// removeRespConnHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func removeResponseConnHeaders(c *app.RequestContext) {
	c.Response.Header.VisitAll(func(k, v []byte) {
		if cast.B2S(k) == "Connection" {
			for _, sf := range strings.Split(cast.B2S(v), ",") {
				if sf = textproto.TrimString(sf); sf != "" {
					c.Response.Header.DelBytes(cast.S2B(sf))
				}
			}
		}
	})
}

// checkTeHeader check RequestHeader if has 'Te: trailers'
// See https://github.com/golang/go/issues/21096
func checkTeHeader(header *protocol.RequestHeader) bool {
	teHeaders := header.PeekAll("Te")
	for _, te := range teHeaders {
		if bytes.Contains(te, []byte("trailers")) {
			return true
		}
	}
	return false
}

func (r *Proxy) defaultErrorHandler(c *app.RequestContext, _ error) {
	c.Response.Header.SetStatusCode(consts.StatusBadGateway)
}

var respTmpHeaderPool = sync.Pool{
	New: func() interface{} {
		return make(map[string][]string)
	},
}

func (r *Proxy) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	req := &ctx.Request
	resp := &ctx.Response

	ctx.Set(config.UPSTREAM_ADDR, r.targetHost)

	// save tmp resp header
	respTmpHeader := respTmpHeaderPool.Get().(map[string][]string)
	if r.saveOriginResHeader {
		resp.Header.SetNoDefaultContentType(true)
		resp.Header.VisitAll(func(key, value []byte) {
			keyStr := string(key)
			valueStr := string(value)
			if _, ok := respTmpHeader[keyStr]; !ok {
				respTmpHeader[keyStr] = []string{valueStr}
			} else {
				respTmpHeader[keyStr] = append(respTmpHeader[keyStr], valueStr)
			}
		})
	}
	if r.director != nil {
		r.director(&ctx.Request)
	}
	req.Header.ResetConnectionClose()

	hasTeTrailer := false
	if r.transferTrailer {
		hasTeTrailer = checkTeHeader(&req.Header)
	}

	removeRequestConnHeaders(ctx)
	// Remove hop-by-hop headers to the backend. Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	for _, h := range hopHeaders {
		if r.transferTrailer && h == "Trailer" {
			continue
		}
		req.Header.DelBytes(cast.S2B(h))
	}

	// Check if 'trailers' exists in te header, If exists, add an additional Te header
	if r.transferTrailer && hasTeTrailer {
		req.Header.Set("Te", "trailers")
	}

	// prepare request(replace headers and some URL host)
	if ip, _, err := net.SplitHostPort(ctx.RemoteAddr().String()); err == nil {
		tmp := req.Header.Peek("X-Forwarded-For")

		if len(tmp) > 0 {
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			buf.Write(tmp)
			buf.WriteString(", ")
			buf.WriteString(ip)
			ip = buf.String()
		}
		if tmp == nil || string(tmp) != "" {
			req.Header.Set("X-Forwarded-For", ip)
		}
	}

	fn := client.Do
	if r.client != nil {
		fn = r.client.Do
	}

	err := fn(c, req, resp)
	if err != nil {
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		buf.Write(req.Method())
		buf.Write(spaceByte)
		buf.Write(req.URI().FullURI())
		uri := buf.String()

		logger := log.FromContext(c)
		logger.ErrorContext(c, "sent upstream error",
			slog.String("error", err.Error()),
			slog.String("upstream", uri),
		)

		if err.Error() == "timeout" {
			ctx.Set("target_timeout", true)
		}

		r.getErrorHandler()(ctx, err)
		return
	}

	// add tmp resp header
	for key, hs := range respTmpHeader {
		for _, h := range hs {
			resp.Header.Add(key, h)
		}
	}

	// Clear and put respTmpHeader back to respTmpHeaderPool
	for k := range respTmpHeader {
		delete(respTmpHeader, k)
	}
	respTmpHeaderPool.Put(respTmpHeader)

	removeResponseConnHeaders(ctx)

	for _, h := range hopHeaders {
		if r.transferTrailer && h == "Trailer" {
			continue
		}
		resp.Header.DelBytes(cast.S2B(h))
	}

	if r.modifyResponse == nil {
		return
	}

	err = r.modifyResponse(resp)
	if err != nil {
		r.getErrorHandler()(ctx, err)
	}

}

// SetDirector use to customize protocol.Request
func (r *Proxy) SetDirector(director func(req *protocol.Request)) {
	r.director = director
}

// SetClient use to customize client
func (r *Proxy) SetClient(client *client.Client) {
	r.client = client
}

// SetModifyResponse use to modify response
func (r *Proxy) SetModifyResponse(mr func(*protocol.Response) error) {
	r.modifyResponse = mr
}

// SetErrorHandler use to customize error handler
func (r *Proxy) SetErrorHandler(eh func(c *app.RequestContext, err error)) {
	r.errorHandler = eh
}

func (r *Proxy) SetTransferTrailer(b bool) {
	r.transferTrailer = b
}

func (r *Proxy) SetSaveOriginResHeader(b bool) {
	r.saveOriginResHeader = b
}

func (r *Proxy) getErrorHandler() func(c *app.RequestContext, err error) {
	if r.errorHandler != nil {
		return r.errorHandler
	}
	return r.defaultErrorHandler
}
