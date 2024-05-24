package gateway

import (
	"regexp"
	"strings"
	"testing"
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

// 方案1：使用正则表达式和 strings.Builder
func replaceVariablesRegex(template string, values map[string]string) string {
	var re = regexp.MustCompile(`\$\w+`)
	matches := re.FindAllString(template, -1)

	var sb strings.Builder
	sb.Grow(len(template) + 100) // 预估大小，避免过多的内存重新分配

	lastIndex := 0
	for _, match := range matches {
		idx := strings.Index(template[lastIndex:], match)
		if idx == -1 {
			continue
		}
		idx += lastIndex

		sb.WriteString(template[lastIndex:idx])
		if val, ok := values[match[1:]]; ok { // 去掉 '$' 符号查找 map
			sb.WriteString(val)
		} else {
			sb.WriteString(match) // 若 map 中不存在该变量，则保留原变量
		}
		lastIndex = idx + len(match)
	}

	// 写入最后一部分
	sb.WriteString(template[lastIndex:])
	return sb.String()
}

// 方案2：使用 strings.NewReplacer
func replaceVariablesReplacer(template string, values map[string]string) string {
	// 创建一个长度为变量数量两倍的 slice，用于 strings.NewReplacer
	replacements := make([]string, 0, len(values)*2)
	for key, value := range values {
		replacements = append(replacements, "$"+key, value)
	}

	// 使用 strings.NewReplacer 创建替换器
	replacer := strings.NewReplacer(replacements...)

	// 执行替换操作
	return replacer.Replace(template)
}

func replaceVars(template string, data map[string]string) string {
	// Convert template string to byte slice
	tplBytes := []byte(template)

	// Compile the regular expression
	re := regexp.MustCompile(`\$(\w+)`)

	// Find all matches of the regular expression
	matches := re.FindAllIndex(tplBytes, -1)

	// Create a new byte slice to construct the replaced template
	newTplBytes := make([]byte, len(tplBytes))
	n := 0 // Index for the new byte slice

	// Iterate over the matches
	for _, match := range matches {
		// Extract the variable name from the match
		varName := string(tplBytes[match[1]:match[2]])

		// Check if the variable name exists in the data map
		if value, ok := data[varName]; ok {
			// Convert the value to a byte slice
			valueBytes := []byte(value)

			// Copy the value bytes to the new byte slice
			copy(newTplBytes[n:], valueBytes)
			n += len(valueBytes)
		} else {
			// If the variable name doesn't exist, use the original substring
			copy(newTplBytes[n:], tplBytes[match[1]:match[2]])
			n += len(tplBytes[match[1]:match[2]])
		}

		// Copy the remaining characters from the template string
		copy(newTplBytes[n:], tplBytes[match[2]:])
		n += len(tplBytes[match[2]:])
	}

	// Convert the new byte slice to a string and return it
	return string(newTplBytes[:n])
}

// 基准测试：replaceVariablesRegex
func BenchmarkReplaceVariablesRegex(b *testing.B) {
	values := map[string]string{
		"time":                   "2024-05-20T15:04:05Z07:00",
		"remote_addr":            "127.0.0.1",
		"request":                "GET / HTTP/1.1",
		"status":                 "200",
		"request_body":           "some body",
		"upstream_addr":          "192.168.0.1",
		"upstream_status":        "502",
		"http_x_forwarded_for":   "10.0.0.1",
		"request_time":           "0.123",
		"upstream_response_time": "0.456",
	}

	for i := 0; i < b.N; i++ {
		replaceVariablesRegex(template, values)
	}
}

// 基准测试：replaceVariablesReplacer
func BenchmarkReplaceVariablesReplacer(b *testing.B) {
	values := map[string]string{
		"time":                   "2024-05-20T15:04:05Z07:00",
		"remote_addr":            "127.0.0.1",
		"request":                "GET / HTTP/1.1",
		"status":                 "200",
		"request_body":           "some body",
		"upstream_addr":          "192.168.0.1",
		"upstream_status":        "502",
		"http_x_forwarded_for":   "10.0.0.1",
		"request_time":           "0.123",
		"upstream_response_time": "0.456",
	}

	for i := 0; i < b.N; i++ {
		replaceVariablesReplacer(template, values)
	}
}

func BenchmarkGoogleReplacer(b *testing.B) {
	values := map[string]string{
		"time":                   "2024-05-20T15:04:05Z07:00",
		"remote_addr":            "127.0.0.1",
		"request":                "GET / HTTP/1.1",
		"status":                 "200",
		"request_body":           "some body",
		"upstream_addr":          "192.168.0.1",
		"upstream_status":        "502",
		"http_x_forwarded_for":   "10.0.0.1",
		"request_time":           "0.123",
		"upstream_response_time": "0.456",
	}

	for i := 0; i < b.N; i++ {
		replaceVars(template, values)
	}
}
