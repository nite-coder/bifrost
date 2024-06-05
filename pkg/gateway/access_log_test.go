package gateway

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/valyala/bytebufferpool"
)

var template = `{"time":"$time",
					"remote_addr":"$remote_addr",
					"request":"$request",
					"status":$status,
					"req_body":"$request_body",
					"upstream_addr":"$upstream_addr",
					"upstream_status":"$upstream_status",
					"x_forwarded_for":"$http_x_forwarded_for",
					"request_time":$request_time,
					"upstream_response_time":$upstream_response_time}`

func replaceTemplate(template string, values map[string]string) string {
	var builder strings.Builder
	builder.Grow(len(template) + 100) // 预估最终字符串长度

	varName := ""
	inVar := false

	for _, char := range template {
		if char == '$' {
			inVar = true
			varName = ""
			continue
		}

		if inVar {
			if char == ' ' || char == ',' || char == '}' || char == '\n' {
				inVar = false
				if value, exists := values[varName]; exists {
					builder.WriteString(value)
				} else {
					builder.WriteString("$" + varName)
				}
				builder.WriteRune(char)
			} else {
				varName += string(char)
			}
		} else {
			builder.WriteRune(char)
		}
	}

	// 处理最后一个变量
	if inVar {
		if value, exists := values[varName]; exists {
			builder.WriteString(value)
		} else {
			builder.WriteString("$" + varName)
		}
	}

	return builder.String()
}

func replaceTemplateWithReplacer(template string, values map[string]string) string {
	replacements := make([]string, 0, len(values)*2)
	for key, value := range values {
		replacements = append(replacements, "$"+key, value)
	}
	replacer := strings.NewReplacer(replacements...)
	return replacer.Replace(template)
}

func BenchmarkReplaceTemplate(b *testing.B) {
	template := `{"time":"$time",
							"remote_addr":"$remote_addr",
							"request":"$request_method $request_uri $request_protocol",
							"req_body":"$request_body",
							"status":$status,
							"upstream_addr":"$upstream_addr",
							"upstream_status":$upstream_status,
							"x_forwarded_for":"$header_X-Forwarded-For",
							"duration":$duration,
							"upstream_duration":$upstream_duration}`
	values := map[string]string{
		"time":                   "2024-06-03T17:17:46Z",
		"remote_addr":            "192.168.1.1",
		"request_method":         "GET",
		"request_uri":            "/index.html",
		"request_protocol":       "HTTP/1.1",
		"request_body":           "",
		"status":                 "200",
		"upstream_addr":          "10.0.0.1:80",
		"upstream_status":        "200",
		"header_X-Forwarded-For": "203.0.113.1",
		"duration":               "0.123",
		"upstream_duration":      "0.456",
	}

	for i := 0; i < b.N; i++ {
		_ = replaceTemplate(template, values)
	}
}

func BenchmarkReplaceTemplateWithReplacer(b *testing.B) {
	template := `{"time":"$time",
							"remote_addr":"$remote_addr",
							"request":"$request_method $request_uri $request_protocol",
							"req_body":"$request_body",
							"status":$status,
							"upstream_addr":"$upstream_addr",
							"upstream_status":$upstream_status,
							"x_forwarded_for":"$header_X-Forwarded-For",
							"duration":$duration,
							"upstream_duration":$upstream_duration}`
	values := map[string]string{
		"time":                   "2024-06-03T17:17:46Z",
		"remote_addr":            "192.168.1.1",
		"request_method":         "GET",
		"request_uri":            "/index.html",
		"request_protocol":       "HTTP/1.1",
		"request_body":           "",
		"status":                 "200",
		"upstream_addr":          "10.0.0.1:80",
		"upstream_status":        "200",
		"header_X-Forwarded-For": "203.0.113.1",
		"duration":               "0.123",
		"upstream_duration":      "0.456",
	}

	for i := 0; i < b.N; i++ {
		_ = replaceTemplateWithReplacer(template, values)
	}
}

var path = []byte("/spot/orders")
var queryString = []byte("a=b&c=d")

func BenchmarkStringBuilder(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var builder strings.Builder
		builder.Write(path)
		if len(queryString) > 0 {
			builder.Write(questionByte)
			builder.Write(queryString)
		}
		_ = builder.String()
	}
}

func BenchmarkBytesBuffer(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var buffer bytes.Buffer
		buffer.Write(path)
		if len(queryString) > 0 {
			buffer.Write(questionByte)
			buffer.Write(queryString)
		}
		_ = buffer.String()
	}
}

func BenchmarkByteBufferPool(b *testing.B) {
	for n := 0; n < b.N; n++ {
		buf := bytebufferpool.Get()
		buf.Write(path)
		if len(queryString) > 0 {
			buf.Write(questionByte)
			buf.Write(queryString)
		}
		_ = buf.String()
		bytebufferpool.Put(buf)
	}
}

var testData = `{
	"market": "BTC_USDT",
	"base": "BTC",
	"quote": "USDT",
	"type": "limit",
	"price": "25000",
	"size": "0.0001",
	"side": "sell",
	"user_id": 1,
	"text": "你好"
}`

func escapeJSONStringBuilder(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		switch c {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '\b':
			b.WriteString("\\b")
		case '\f':
			b.WriteString("\\f")
		default:
			if c < 32 {
				b.WriteString("\\u")
				b.WriteString(strconv.FormatUint(uint64(c), 16))
			} else {
				r, size := utf8.DecodeRuneInString(s[i:])
				b.WriteRune(r)
				i += size - 1
			}
		}
		i++
	}
	return b.String()
}

// escapeJSON escapes special characters in a JSON string
func escapeJSON4(input string) string {
	var builder strings.Builder
	for _, char := range input {
		switch char {
		case '\\':
			builder.WriteString("\\\\")
		case '"':
			builder.WriteString("\\\"")
		case '\b':
			builder.WriteString("\\b")
		case '\f':
			builder.WriteString("\\f")
		case '\n':
			builder.WriteString("\\n")
		case '\r':
			builder.WriteString("\\r")
		case '\t':
			builder.WriteString("\\t")
		default:
			if char < 0x20 {
				builder.WriteString("\\u")
				builder.WriteString("00")
				builder.WriteString(strings.ToUpper(string("0123456789ABCDEF"[char>>4])))
				builder.WriteString(strings.ToUpper(string("0123456789ABCDEF"[char&0xF])))
			} else {
				builder.WriteRune(char)
			}
		}
	}
	return builder.String()
}

func escapeJSONBytePool(s string) string {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	for i := 0; i < len(s); {
		c := s[i]
		switch c {
		case '"':
			buf.WriteString("\\\"")
		case '\\':
			buf.WriteString("\\\\")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		case '\b':
			buf.WriteString("\\b")
		case '\f':
			buf.WriteString("\\f")
		default:
			if c < 32 {
				buf.WriteString("\\u")
				// Use a fixed-size array to avoid further allocations
				tmp := [4]byte{}
				hex := strconv.AppendUint(tmp[:0], uint64(c), 16)
				if len(hex) == 1 {
					buf.WriteString("000")
				} else if len(hex) == 2 {
					buf.WriteString("00")
				} else if len(hex) == 3 {
					buf.WriteString("0")
				}
				buf.Write(hex)
			} else {
				r, size := utf8.DecodeRuneInString(s[i:])
				runeBytes := make([]byte, 4)
				n := utf8.EncodeRune(runeBytes, r)
				buf.Write(runeBytes[:n])
				i += size - 1
			}
		}
		i++
	}
	return buf.String()
}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}

	buf := bytebufferpool.Get()
	buf.Write(b)
	return buf.String()
}

func sonicEscape(i string) string {
	b, err := sonic.Marshal(i)
	if err != nil {
		return i
	}

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, _ = buf.Write(b)
	return buf.String()
}

func escapeJSON2(s string) string {
	var escaped bytes.Buffer
	escaped.Grow(len(s) + len(s)/4) // 预先分配足够大的缓冲区
	last := 0
	for i := 0; i < len(s); i++ {
		if !isSafePathKeyChar(s[i]) {
			escaped.WriteString(s[last:i])
			escaped.WriteByte('\\')
			escaped.WriteByte(s[i])
			last = i + 1
		}
	}
	escaped.WriteString(s[last:])
	return escaped.String()
}

func TestEscapeJSON1(t *testing.T) {
	fmt.Println(escapeJSON2(testData))
}

func BenchmarkEscapeJSONStringBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = escapeJSONStringBuilder(testData)
	}
}

func BenchmarkEscapeJSON2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = escapeJSON2(testData)
	}
}

func BenchmarkEscapeJSON1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = escapeJSON(testData)
	}
}

func BenchmarkDirectWrite(b *testing.B) {
	logFile, err := os.Create("direct_write.log")
	if err != nil {
		b.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	writer := bufio.NewWriterSize(logFile, 4096)
	defer writer.Flush()

	for i := 0; i < b.N; i++ {
		_, err := writer.WriteString(`{"time":"2024-06-04T11:32:38", "remote_addr":"127.0.0.1", "request":"POST /spot/orders?a=b HTTP/1.1", "req_body":""{\"market\":\"BTC_USDT\",\"base\":\"BTC\",\"quote\":\"USDT\",\"type\":\"limit\",\"price\":\"25000\",\"size\":\"0.0001\",\"side\":\"sell\",\"user_id\":1,\"text\":\"你好世界\"}"", "status":200, "upstream_addr":"127.0.0.1:8000", "upstream_status":200, "x_forwarded_for":"", "duration":0.00039, "upstream_duration":0.000264}` + "\n")
		if err != nil {
			b.Fatalf("Failed to write log entry: %v", err)
		}
	}
}

func BenchmarkBatchWrite(b *testing.B) {

	logFile, err := os.Create("batch_write.log")
	if err != nil {
		b.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	writer := bufio.NewWriterSize(logFile, 4096)
	defer writer.Flush()

	batchSize := 100
	buffer := make([]string, 0, batchSize)

	for i := 0; i < b.N; i++ {
		buffer = append(buffer, `{"time":"2024-06-04T11:32:38", "remote_addr":"127.0.0.1", "request":"POST /spot/orders?a=b HTTP/1.1", "req_body":""{\"market\":\"BTC_USDT\",\"base\":\"BTC\",\"quote\":\"USDT\",\"type\":\"limit\",\"price\":\"25000\",\"size\":\"0.0001\",\"side\":\"sell\",\"user_id\":1,\"text\":\"你好世界\"}"", "status":200, "upstream_addr":"127.0.0.1:8000", "upstream_status":200, "x_forwarded_for":"", "duration":0.00039, "upstream_duration":0.000264}`)
		if len(buffer) >= batchSize {
			_, err := writer.WriteString(strings.Join(buffer, "\n"))
			if err != nil {
				b.Fatalf("Failed to write log entry: %v", err)
			}
			buffer = buffer[:0]
		}
	}

	// Flush remaining entries in buffer
	if len(buffer) > 0 {
		_, err := writer.WriteString(strings.Join(buffer, "\n"))
		if err != nil {
			b.Fatalf("Failed to write log entry: %v", err)
		}
	}
}
