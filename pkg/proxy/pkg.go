package proxy

import (
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/valyala/bytebufferpool"
)

var (
	spaceByte = []byte{byte(' ')}
)

// IsASCIIPrint returns whether s is ASCII and printable according to
// https://tools.ietf.org/html/rfc20#section-4.2.
func IsASCIIPrint(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
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
