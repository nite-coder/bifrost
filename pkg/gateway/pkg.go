package gateway

import (
	"http-benchmark/pkg/middleware/addprefix"
	"http-benchmark/pkg/middleware/replacepath"
	"http-benchmark/pkg/middleware/replacepathregex"
	"http-benchmark/pkg/middleware/stripprefix"
	"unsafe"

	"github.com/cloudwego/hertz/pkg/app"
)

var (
	spaceByte    = []byte{byte(' ')}
	questionByte = []byte{byte('?')}
)

// b2s converts byte slice to a string without memory allocation.
// See https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ .
func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// s2b converts string to a byte slice without memory allocation.
func s2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func init() {
	_ = RegisterMiddleware("strip_prefix", func(params map[string]any) (app.HandlerFunc, error) {
		val := params["prefixes"].([]any)

		prefixes := make([]string, 0)
		for _, v := range val {
			prefixes = append(prefixes, v.(string))
		}

		m := stripprefix.NewMiddleware(prefixes)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("add_prefix", func(params map[string]any) (app.HandlerFunc, error) {
		prefix := params["prefix"].(string)
		m := addprefix.NewMiddleware(prefix)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("replace_path", func(params map[string]any) (app.HandlerFunc, error) {
		newPath := params["path"].(string)
		m := replacepath.NewMiddleware(newPath)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("replace_path_regex", func(params map[string]any) (app.HandlerFunc, error) {
		regex := params["regex"].(string)
		replacement := params["replacement"].(string)
		m := replacepathregex.NewMiddleware(regex, replacement)
		return m.ServeHTTP, nil
	})
}
