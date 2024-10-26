package accesslog

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/nite-coder/bifrost/pkg/config"
)

var (
	content = `{"label": "hello 您好 ~"}`
)

func TestEscape(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		escapeType config.EscapeType
		expected   string
	}{
		{"empty string", "", config.DefaultEscape, ""},
		{"default escape", "hello 您好", config.DefaultEscape, `hello \xe6\x82\xa8\xe5\xa5\xbd`},
		{"json escape", content, config.JSONEscape, `{\"label\": \"hello 您好 ~\"}`},
		{"none escape", content, config.NoneEscape, content},
		{"invalid escape", content, " invalid", content},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := escape(tt.input, tt.escapeType)
			if actual != tt.expected {
				t.Errorf("escape(%q, %s) = %s, want %s", tt.input, tt.escapeType, actual, tt.expected)
			}
		})
	}
}

func escapeJSONSonic(comp string) string {
	b, _ := sonic.Marshal(comp)
	return string(b[1 : len(b)-1])
}

func BenchmarkEscapeJSON(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"NoEscape", "This is a normal string without any special characters."},
		{"WithEscape", "This string has \"quotes\" and \\ backslashes and \n newlines."},
		{"AllEscape", "\"\\\n\r\t\b\f"},
		{"WithChinese", `{
			"market": "BTC_USDT",
			"base": "BTC",
			"quote": "USDT",
			"type": "limit",
			"price": "25000",
			"size": "0.0001",
			"side": "sell",
			"user_id": 1,
			"text": "你好世界"
		}`},
	}

	for _, tc := range testCases {
		b.Run("Sonic_"+tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				escapeJSONSonic(tc.input)
			}
		})

		b.Run("Custom_"+tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				escapeJSON(tc.input)
			}
		})
	}
}
