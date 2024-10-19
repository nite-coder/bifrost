package runtime

import (
	"fmt"
	"runtime"
	"strings"
)

func StackTrace() string {
	stackBuf := make([]uintptr, 50)
	length := runtime.Callers(3, stackBuf)
	stack := stackBuf[:length]

	var b strings.Builder
	frames := runtime.CallersFrames(stack)

	for {
		frame, more := frames.Next()

		if !strings.Contains(frame.File, "runtime/") {
			_, _ = b.WriteString(fmt.Sprintf("\n\tFile: %s, Line: %d. Function: %s", frame.File, frame.Line, frame.Function))
		}

		if !more {
			break
		}
	}
	return b.String()
}
