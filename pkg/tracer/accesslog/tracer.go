package accesslog

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/valyala/bytebufferpool"
)

type Tracer struct {
	opts      config.AccessLogOptions
	matchVars []string
	writer    *BufferedLogger
}

func NewTracer(opts config.AccessLogOptions) (*Tracer, error) {
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.RFC3339
	}

	words := strings.Fields(opts.Template)
	opts.Template = strings.Join(words, " ") + "\n"

	bufferedLogger, err := NewBufferedLogger(opts)
	if err != nil {
		return nil, err
	}

	tracer := &Tracer{
		opts:      opts,
		matchVars: parseVariables(opts.Template),
		writer:    bufferedLogger,
	}

	return tracer, nil
}

func (t *Tracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	return ctx
}

func (t *Tracer) Finish(ctx context.Context, c *app.RequestContext) {
	vals := t.buildReplacer(c)
	if vals == nil {
		return
	}

	replacer := strings.NewReplacer(vals...)
	result := replacer.Replace(t.opts.Template)
	t.writer.Write(result)
}

func (t *Tracer) Close() error {
	if strings.EqualFold(t.opts.Output, "stderr") {
		return nil
	}

	err := t.writer.Close()
	if err != nil {
		slog.Debug("failed to close access log writer", "error", err)
	}
	return err
}

func (t *Tracer) buildReplacer(c *app.RequestContext) []string {
	httpStats := c.GetTraceInfo().Stats()
	httpStart := httpStats.GetEvent(stats.HTTPStart)
	if httpStart == nil {
		// if the request is closed, the `Finish` is called, and HTTPStart event will be nil
		return nil
	}

	httpFinish := httpStats.GetEvent(stats.HTTPFinish)
	statErr := httpStats.Error()

	timeNow := timecache.Now()
	contentType := c.Request.Header.ContentType()
	replacements := make([]string, 0, len(t.matchVars)*2)

	for _, matchVal := range t.matchVars {
		switch matchVal {
		case config.TIME:
			now := timeNow.Format(t.opts.TimeFormat)
			replacements = append(replacements, config.TIME, now)
		case config.REMOTE_ADDR:
			var ip string
			switch addr := c.RemoteAddr().(type) {
			case *net.UDPAddr:
				ip = addr.IP.String()
			case *net.TCPAddr:
				ip = addr.IP.String()
			}
			replacements = append(replacements, config.REMOTE_ADDR, ip)
		case config.HOST:
			val, _ := variable.Get(config.HOST, c)
			host, _ := cast.ToString(val)
			replacements = append(replacements, config.HOST, host)
		case config.REQUEST_METHOD:
			replacements = append(replacements, config.REQUEST_METHOD, cast.B2S(c.Request.Method()))
		case config.REQUEST_URI:
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			val, found := c.Get(config.REQUEST_PATH)
			if found {
				b, ok := val.([]byte)
				if ok {
					_, _ = buf.Write(b)
					if len(c.Request.QueryString()) > 0 {
						_, _ = buf.Write(questionByte)
						_, _ = buf.Write(c.Request.QueryString())
					}

					replacements = append(replacements, config.REQUEST_URI, buf.String())
				}
				continue
			}

			_, _ = buf.Write(c.Request.Path())
			if len(c.Request.QueryString()) > 0 {
				_, _ = buf.Write(questionByte)
				_, _ = buf.Write(c.Request.QueryString())
			}
			replacements = append(replacements, config.REQUEST_URI, buf.String())

		case config.REQUEST_PATH:
			val, _ := variable.Get(config.REQUEST_PATH, c)
			path, _ := cast.ToString(val)
			replacements = append(replacements, config.REQUEST_PATH, path)
		case config.REQUEST_PROTOCOL:
			replacements = append(replacements, config.REQUEST_PROTOCOL, c.Request.Header.GetProtocol())
		case config.REQUEST_BODY:
			// if content type is grpc, the $request_body will be ignored
			if bytes.Equal(contentType, grpcContentType) {
				replacements = append(replacements, config.REQUEST_BODY, "")
				continue
			}

			body := escape(cast.B2S(c.Request.Body()), t.opts.Escape)
			replacements = append(replacements, config.REQUEST_BODY, body)
		case config.STATUS:
			status := strconv.Itoa(c.Response.StatusCode())

			// this case for http2 client disconnected
			if statErr != nil && statErr.Error() == "client disconnected" {
				status = strconv.Itoa(499)
			}

			replacements = append(replacements, config.STATUS, status)
		case config.UPSTREAM_PROTOCOL:
			replacements = append(replacements, config.UPSTREAM_PROTOCOL, c.Request.Header.GetProtocol())
		case config.UPSTREAM_METHOD:
			replacements = append(replacements, config.UPSTREAM_METHOD, cast.B2S(c.Request.Method()))
		case config.UPSTREAM_URI:
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			_, _ = buf.Write(c.Request.Path())

			if len(c.Request.QueryString()) > 0 {
				_, _ = buf.Write(questionByte)
				_, _ = buf.Write(c.Request.QueryString())
			}

			replacements = append(replacements, config.UPSTREAM_URI, buf.String())
		case config.UPSTREAM_PATH:
			replacements = append(replacements, config.UPSTREAM_PATH, cast.B2S(c.Request.Path()))
		case config.UPSTREAM_ADDR:
			addr := c.GetString(config.UPSTREAM_ADDR)
			replacements = append(replacements, config.UPSTREAM_ADDR, addr)
		case config.UPSTREAM_STATUS:
			code := c.GetInt(config.UPSTREAM_STATUS)
			replacements = append(replacements, config.UPSTREAM_STATUS, strconv.Itoa(code))
		case config.UPSTREAM_DURATION:
			dur := c.GetString(config.UPSTREAM_DURATION)
			if len(dur) == 0 {
				dur = "0"
			}
			replacements = append(replacements, config.UPSTREAM_DURATION, dur)
		case config.DURATION:
			dur := httpFinish.Time().Sub(httpStart.Time()).Microseconds()
			duration := strconv.FormatFloat(float64(dur)/1e6, 'f', -1, 64)
			replacements = append(replacements, config.DURATION, duration)
		case config.RECEIVED_SIZE:
			replacements = append(replacements, config.RECEIVED_SIZE, strconv.Itoa(httpStats.RecvSize()))
		case config.SEND_SIZE:
			replacements = append(replacements, config.SEND_SIZE, strconv.Itoa(httpStats.SendSize()))
		case config.TRACE_ID:
			traceID := c.GetString(config.TRACE_ID)
			replacements = append(replacements, config.TRACE_ID, traceID)
		case config.GRPC_STATUS:
			status := ""

			val, found := c.Get(config.GRPC_STATUS)
			if found {
				status, _ = cast.ToString(val)
			}

			replacements = append(replacements, config.GRPC_STATUS, status)
		case config.GRPC_MESSAGE:
			grpcMessage := c.GetString(config.GRPC_MESSAGE)
			replacements = append(replacements, config.GRPC_MESSAGE, grpcMessage)
		default:

			if strings.HasPrefix(matchVal, "$upstream_header_") {
				headerVal := matchVal[len("$upstream_header_"):]
				headerVal = c.Response.Header.Get(headerVal)
				headerVal = escape(headerVal, t.opts.Escape)
				replacements = append(replacements, matchVal, headerVal)
				continue
			}

			if strings.HasPrefix(matchVal, "$header_") {
				headerVal := matchVal[len("$header_"):]

				if headerVal == "X-Forwarded-For" {
					ip := c.GetString("X-Forwarded-For")
					replacements = append(replacements, matchVal, ip)
					continue
				}

				headerVal = c.Request.Header.Get(headerVal)
				headerVal = escape(headerVal, t.opts.Escape)
				replacements = append(replacements, matchVal, headerVal)
				continue
			}

			val, found := c.Get(matchVal)
			if found {
				s, _ := val.(string)
				replacements = append(replacements, matchVal, s)
				continue
			}

			replacements = append(replacements, matchVal, matchVal)
		}
	}

	return replacements
}

func escape(s string, escapeType config.EscapeType) string {
	if len(s) == 0 {
		return s
	}

	switch escapeType {
	case config.DefaultEscape:
		s = escapeString(s)
	case config.JSONEscape:
		s = escapeJSON(s)
	case config.NoneEscape, "":
		return s
	default:
		return s
	}

	return s
}

// escapeString function to escape special characters
func escapeString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		if c == '"' || c == '\\' || c < 32 || c > 126 {
			b.WriteString(`\x`)
			b.WriteString(strconv.FormatUint(uint64(c), 16))
			i++
		} else {
			r, size := utf8.DecodeRuneInString(s[i:])
			b.WriteRune(r)
			i += size
		}
	}
	return b.String()
}

// For json escaping, all characters not allowed in JSON strings will be escaped: characters “"” and “\” are escaped as “\"” and “\\”,
// characters with values less than 32 are escaped as “\n”, “\r”, “\t”, “\b”, “\f”, or “\u00XX”.
func escapeJSON(comp string) string {
	for i := 0; i < len(comp); i++ {
		if needsEscape(comp[i]) {
			ncomp := make([]byte, 0, len(comp)*2) // allocate enough space
			ncomp = append(ncomp, comp[:i]...)

			for ; i < len(comp); i++ {
				switch comp[i] {
				case '"':
					ncomp = append(ncomp, '\\', '"')
				case '\\':
					ncomp = append(ncomp, '\\', '\\')
				case '\n':
					ncomp = append(ncomp, '\\', 'n')
				case '\r':
					ncomp = append(ncomp, '\\', 'r')
				case '\t':
					ncomp = append(ncomp, '\\', 't')
				case '\b':
					ncomp = append(ncomp, '\\', 'b')
				case '\f':
					ncomp = append(ncomp, '\\', 'f')
				default:
					if comp[i] < 32 {
						ncomp = append(ncomp, '\\', 'u', '0', '0',
							hexChars[comp[i]>>4], hexChars[comp[i]&0xF])
					} else {
						ncomp = append(ncomp, comp[i])
					}
				}
			}
			return string(ncomp)
		}
	}
	return comp
}

func needsEscape(c byte) bool {
	return c == '"' || c == '\\' || c < 32
}

var hexChars = []byte("0123456789ABCDEF")
