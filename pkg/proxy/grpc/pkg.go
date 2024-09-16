package grpc

import (
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/valyala/bytebufferpool"
)

var (
	spaceByte = []byte{byte(' ')}
)

func fullURI(req *protocol.Request) string {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, _ = buf.Write(req.Method())
	_, _ = buf.Write(spaceByte)
	_, _ = buf.Write(req.URI().FullURI())
	return buf.String()
}
