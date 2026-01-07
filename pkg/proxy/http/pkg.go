package http

import (
	"net/url"
	"strings"

	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/valyala/bytebufferpool"
)

var (
	spaceByte       = []byte{byte(' ')}
	https           = []byte("https")
	trailersBytes   = []byte("trailers")
	chunkedTransfer = false
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
			target = string(req.Host()) + target
		} else {
			target = string(req.Host()) + "/" + target
		}
		bslash = strings.HasSuffix(target, "/")
	}

	targetQuery := strings.Split(target, "?")
	buffer := bytebufferpool.Get()
	defer bytebufferpool.Put(buffer)

	_, _ = buffer.WriteString(targetQuery[0])

	escapedPath := (&url.URL{Path: string(req.URI().Path())}).EscapedPath()
	switch {
	case aslash && bslash:
		if len(escapedPath) > 1 {
			_, _ = buffer.WriteString(escapedPath[1:])
		}
	case !aslash && !bslash:
		_, _ = buffer.Write([]byte{'/'})
		_, _ = buffer.WriteString(escapedPath)
	default:
		_, _ = buffer.WriteString(escapedPath)
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

func fullURI(req *protocol.Request) string {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, _ = buf.Write(req.Method())
	_, _ = buf.Write(spaceByte)
	_, _ = buf.Write(req.URI().FullURI())
	return buf.String()
}
