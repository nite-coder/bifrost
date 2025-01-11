package accesslog

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/variable"
)

type BufferedLogger struct {
	writer     *bufio.Writer
	file       *os.File
	mu         sync.Mutex
	flushTimer *time.Timer
	options    *config.AccessLogOptions
}

func NewBufferedLogger(opts config.AccessLogOptions) (*BufferedLogger, error) {

	var writer io.Writer

	logger := &BufferedLogger{
		options: &opts,
	}

	output := strings.ToLower(opts.Output)
	switch output {
	case "":
		writer = io.Discard
	case "stderr":
		writer = os.Stderr
	default:
		file, err := os.OpenFile(opts.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		logger.file = file
		writer = file
	}

	if opts.BufferSize <= 0 {
		opts.BufferSize = 64 * variable.KB
	}

	logger.writer = bufio.NewWriterSize(writer, opts.BufferSize)

	if opts.Flush.Seconds() > 0 {
		logger.flushTimer = time.AfterFunc(opts.Flush, logger.periodicFlush)
	}

	return logger, nil
}

func (l *BufferedLogger) Write(log string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, _ = l.writer.WriteString(log)
}

func (l *BufferedLogger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := l.writer.Flush()
	if err != nil {
		return err
	}

	err = l.file.Sync() // sync to disk
	if err != nil {
		return err
	}

	return nil
}

func (l *BufferedLogger) periodicFlush() {
	_ = l.Flush()
	l.flushTimer.Reset(l.options.Flush)
}

func (l *BufferedLogger) Close() error {
	if l.flushTimer != nil {
		l.flushTimer.Stop()
	}

	_ = l.Flush()

	if l.file != nil {
		return l.file.Close()
	}

	return nil
}
