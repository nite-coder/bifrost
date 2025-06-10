package accesslog

import (
	"bytes"
	"context"
	"log/slog"
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
)

var (
	grpcContentType = []byte("application/grpc")
)

type Tracer struct {
	writer     *BufferedLogger
	options    config.AccessLogOptions
	directives []string
}

func NewTracer(opts config.AccessLogOptions) (*Tracer, error) {
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.DateTime
	}
	words := strings.Fields(opts.Template)
	opts.Template = strings.Join(words, " ") + "\n"
	bufferedLogger, err := NewBufferedLogger(opts)
	if err != nil {
		return nil, err
	}
	tracer := &Tracer{
		options:    opts,
		directives: variable.ParseDirectives(opts.Template),
		writer:     bufferedLogger,
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
	result := replacer.Replace(t.options.Template)
	t.writer.Write(result)
}
func (t *Tracer) Close() error {
	if strings.EqualFold(t.options.Output, "stderr") {
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
	replacements := make([]string, 0, len(t.directives)*2)
	for _, key := range t.directives {
		switch key {
		case variable.Time:
			timeNow := timecache.Now()
			now := timeNow.Format(t.options.TimeFormat)
			replacements = append(replacements, variable.Time, now)
		case variable.HTTPRequestBody:
			contentType := c.Request.Header.ContentType()
			// if content type is grpc, the $request_body will be ignored
			if bytes.Equal(contentType, grpcContentType) {
				replacements = append(replacements, variable.HTTPRequestBody, "")
				continue
			}
			body := escape(cast.B2S(c.Request.Body()), t.options.Escape)
			replacements = append(replacements, variable.HTTPRequestBody, body)
		case variable.HTTPResponseStatusCode:
			status := strconv.Itoa(c.Response.StatusCode())
			// this case for http2 client disconnected
			statErr := httpStats.Error()
			if statErr != nil && statErr.Error() == "client disconnected" {
				status = strconv.Itoa(499)
			}
			replacements = append(replacements, variable.HTTPResponseStatusCode, status)
		default:
			val := variable.GetString(key, c)
			replacements = append(replacements, key, val)
			continue
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
