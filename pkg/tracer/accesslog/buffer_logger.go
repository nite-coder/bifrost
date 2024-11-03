package accesslog

import (
	"bufio"
	"os"
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
		opts.BufferSize = 64 * variable.KB
	}

	writer := bufio.NewWriterSize(logFile, opts.BufferSize)

	if opts.Flush.Seconds() <= 0 {
		opts.Flush = 1 * time.Minute
	}

	logger := &BufferedLogger{
		writer:  writer,
		options: &opts,
	}

	logger.flushTimer = time.AfterFunc(opts.Flush, logger.periodicFlush)

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
	l.flushTimer.Stop()
	_ = l.Flush()
	return l.file.Close()
}
