package accesslog

import (
	"bufio"
	"context"
	"http-benchmark/pkg/config"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/valyala/bytebufferpool"
)

type Tracer struct {
	opts      config.AccessLogOptions
	matchVars []string
	logChan   chan []string
	logFile   *os.File
	writer    *bufio.Writer
}

func NewTracer(opts config.AccessLogOptions) (*Tracer, error) {
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.RFC3339
	}

	words := strings.Fields(opts.Template)
	opts.Template = strings.Join(words, " ") + "\n"

	var err error
	var logFile *os.File

	switch opts.Output {
	case "stderr", "":
		logFile = os.Stderr
	default:
		logFile, err = os.OpenFile(opts.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	}

	if opts.BufferSize <= 0 {
		opts.BufferSize = 64 * 1024
	}

	writer := bufio.NewWriterSize(logFile, opts.BufferSize)

	if opts.Flush.Seconds() <= 0 {
		opts.Flush = 1 * time.Second
	}

	tracer := &Tracer{
		opts:      opts,
		logChan:   make(chan []string, 1000000),
		matchVars: parseVariables(opts.Template),
		logFile:   logFile,
		writer:    writer,
	}

	go func(t *Tracer) {
		flushTimer := time.NewTimer(opts.Flush)

		for {
			flushTimer.Reset(opts.Flush)

			select {
			case server, ok := <-t.logChan:
				if !ok {
					// Channel closed, flush remaining data
					_ = writer.Flush()
					_ = t.logFile.Sync()
					return
				}

				replacer := strings.NewReplacer(server...)
				result := replacer.Replace(opts.Template)
				_, _ = writer.WriteString(result)
			case <-flushTimer.C:
				_ = writer.Flush()
				_ = t.logFile.Sync()
			}
		}
	}(tracer)

	return tracer, nil
}

func (t *Tracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	return ctx
}

func (t *Tracer) Finish(ctx context.Context, c *app.RequestContext) {
	result := t.buildReplacer(c)
	if result == nil {
		return
	}

	select {
	case t.logChan <- result:
	case <-time.After(1 * time.Second):
		slog.Info("access log queue is full", "length", len(t.logChan))
	}
}

func (t *Tracer) Shutdown() {
	close(t.logChan)
	t.writer.Flush()
	_ = t.logFile.Sync()
	t.logFile.Close()
}

func (t *Tracer) buildReplacer(c *app.RequestContext) []string {
	// TODO: there is a weird request without any information...
	// therefore, we try to get trace info to ensure the request is real
	httpStart := c.GetTraceInfo().Stats().GetEvent(stats.HTTPStart)
	if httpStart == nil {
		return nil
	}

	replacements := make([]string, 0, len(t.matchVars)*2)

	info := c.GetTraceInfo().Stats()

	for _, matchVal := range t.matchVars {
		switch matchVal {
		case config.TIME:
			startTime := httpStart.Time()
			replacements = append(replacements, config.TIME, startTime.Format(t.opts.TimeFormat))
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
			host := c.GetString(config.HOST)
			replacements = append(replacements, config.HOST, host)
		case config.REQUEST_METHOD:
			replacements = append(replacements, config.REQUEST_METHOD, cast.B2S(c.Request.Method()))
		case config.REQUEST_URI:
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			val, found := c.Get(config.REQUEST_PATH)
			if found {
				path, ok := val.(string)
				if ok {
					_, _ = buf.WriteString(path)
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
			val, found := c.Get(config.REQUEST_PATH)
			if found {
				b, ok := val.([]byte)
				if ok {
					replacements = append(replacements, config.REQUEST_PATH, cast.B2S(b))
					continue
				} else {
					replacements = append(replacements, config.REQUEST_PATH, "")
				}
				continue
			}
			replacements = append(replacements, config.REQUEST_PATH, cast.B2S(c.Request.Path()))
		case config.REQUEST_PROTOCOL:
			replacements = append(replacements, config.REQUEST_PROTOCOL, c.Request.Header.GetProtocol())
		case config.REQUEST_BODY:
			body := escape(cast.B2S(c.Request.Body()), t.opts.Escape)
			replacements = append(replacements, config.REQUEST_BODY, body)
		case config.STATUS:
			replacements = append(replacements, config.STATUS, strconv.Itoa(c.Response.StatusCode()))
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
			replacements = append(replacements, config.UPSTREAM_DURATION, c.GetString(config.UPSTREAM_DURATION))
		case config.DURATION:
			val, found := c.Get(config.CLIENT_CANCELED_AT)

			if found {
				cancelTime := val.(time.Time)
				dur := cancelTime.Sub(httpStart.Time()).Microseconds()
				duration := strconv.FormatFloat(float64(dur)/1e6, 'f', -1, 64)
				replacements = append(replacements, config.DURATION, duration)
				continue
			}

			dur := time.Since(httpStart.Time()).Microseconds()
			duration := strconv.FormatFloat(float64(dur)/1e6, 'f', -1, 64)
			replacements = append(replacements, config.DURATION, duration)
		case config.RECEIVED_SIZE:
			replacements = append(replacements, config.RECEIVED_SIZE, strconv.Itoa(info.RecvSize()))
		case config.SEND_SIZE:
			replacements = append(replacements, config.SEND_SIZE, strconv.Itoa(info.SendSize()))
		case config.TRACE_ID:
			traceID := c.GetString(config.TRACE_ID)
			replacements = append(replacements, config.TRACE_ID, traceID)
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

// escapeJSON function to escape characters for JSON strings
func escapeJSON(comp string) string {
	for i := 0; i < len(comp); i++ {
		if !isSafePathKeyChar(comp[i]) {
			ncomp := make([]byte, len(comp)+1)
			copy(ncomp, comp[:i])
			ncomp = ncomp[:i]
			for ; i < len(comp); i++ {
				if !isSafePathKeyChar(comp[i]) {
					ncomp = append(ncomp, '\\')
				}
				ncomp = append(ncomp, comp[i])
			}
			return string(ncomp)
		}
	}
	return comp
}

// isSafePathKeyChar returns true if the input character is safe for not
// needing escaping.
func isSafePathKeyChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c <= ' ' || c > '~' || c == '_' ||
		c == '-' || c == ':'
}
