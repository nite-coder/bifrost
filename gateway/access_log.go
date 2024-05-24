package gateway

import (
	"bufio"
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

type LoggerTracer struct {
	opts      AccessLogOptions
	matchVars []string
	logChan   chan string
	logFile   *os.File
	writer    *bufio.Writer
}

func NewLoggerTracer(opts AccessLogOptions) (*LoggerTracer, error) {

	if opts.TimeFormat == "" {
		opts.TimeFormat = time.RFC3339
	}

	words := strings.Fields(opts.Template)
	opts.Template = strings.Join(words, " ") + "\n"

	logFile, err := os.OpenFile(opts.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	if opts.BufferSize <= 0 {
		opts.BufferSize = 64 * 1024
	}

	writer := bufio.NewWriterSize(logFile, opts.BufferSize)

	if opts.Flush.Seconds() <= 0 {
		opts.Flush = 1 * time.Second
	}

	tracer := &LoggerTracer{
		opts:      opts,
		logChan:   make(chan string, 1000000),
		matchVars: parseVariables(opts.Template),
		logFile:   logFile,
		writer:    writer,
	}

	go func(t *LoggerTracer) {
		flushTimer := time.NewTimer(opts.Flush)

		for {
			flushTimer.Reset(opts.Flush)

			select {
			case entry, ok := <-t.logChan:
				if !ok {
					// Channel closed, flush remaining data
					writer.Flush()
					return
				}
				_, err = writer.WriteString(entry)
				if err != nil {
					if os.IsNotExist(err) || err.Error() == "file already closed" {
						// 尝试重新打开文件
						t.logFile, err = os.OpenFile(t.opts.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
						if err != nil {
							continue
						}
						// 重新写入数据
						t.writer = bufio.NewWriterSize(logFile, opts.BufferSize)
						_, _ = t.writer.WriteString(entry)
					}
				}

			case <-flushTimer.C:
				_ = writer.Flush()
				_ = t.logFile.Sync()
			}
		}
	}(tracer)

	return tracer, nil
}

func (t *LoggerTracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	time := time.Now().UTC()
	c.Set(TIME, time)
	return ctx
}

func (t *LoggerTracer) Finish(ctx context.Context, c *app.RequestContext) {
	replacer := t.buildReplacer(c)

	result := replacer.Replace(t.opts.Template)

	t.logChan <- result
}

func (t *LoggerTracer) Shutdown() {
	close(t.logChan)
	t.writer.Flush()
	_ = t.logFile.Sync()
	t.logFile.Close()
}

func (t *LoggerTracer) buildReplacer(c *app.RequestContext) *strings.Replacer {

	replacements := make([]string, 0, len(t.matchVars)*2)

	for _, matchVal := range t.matchVars {
		switch matchVal {
		case TIME:
			startTime := c.GetTime(TIME)
			replacements = append(replacements, TIME, startTime.Format(t.opts.TimeFormat))
		case REMOTE_ADDR:
			var ip string
			switch addr := c.RemoteAddr().(type) {
			case *net.UDPAddr:
				ip = addr.IP.String()
			case *net.TCPAddr:
				ip = addr.IP.String()
			}
			replacements = append(replacements, REMOTE_ADDR, ip)
		case REQUEST_METHOD:
			replacements = append(replacements, REQUEST_METHOD, b2s(c.Request.Method()))
		case REQUEST_PATH:
			requestPath := c.GetString(REQUEST_PATH)
			replacements = append(replacements, REQUEST_PATH, requestPath)
		case REQUEST_PROTOCOL:
			replacements = append(replacements, REQUEST_PROTOCOL, c.Request.Header.GetProtocol())
		case REQUEST_BODY:
			body := ""
			if len(c.Request.Body()) > 0 {
				body = b2s(c.Request.Body())
				if t.opts.Escape {
					body = escapeString(body)
				}
			}
			replacements = append(replacements, REQUEST_BODY, body)
		case STATUS:
			replacements = append(replacements, STATUS, strconv.Itoa(c.Response.StatusCode()))
		case UPSTREAM_ADDR:
			aa := c.GetString(UPSTREAM_ADDR)
			replacements = append(replacements, UPSTREAM_ADDR, aa)
		case UPSTREAM_RESPONSE_TIME:
			replacements = append(replacements, UPSTREAM_RESPONSE_TIME, c.GetString(UPSTREAM_RESPONSE_TIME))
		case UPSTREAM_STATUS:
			replacements = append(replacements, UPSTREAM_STATUS, strconv.Itoa(c.GetInt(UPSTREAM_STATUS)))
		case Duration:
			dur := time.Since(c.GetTime(TIME)).Microseconds()
			duration := strconv.FormatFloat(float64(dur)/1e6, 'f', -1, 64)
			replacements = append(replacements, Duration, duration)
		default:
			replacements = append(replacements, "$"+matchVal, "$"+matchVal)
		}
	}

	replacer := strings.NewReplacer(replacements...)

	return replacer
}
