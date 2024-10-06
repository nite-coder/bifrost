package accesslog

import (
	"testing"

	"github.com/nite-coder/bifrost/pkg/config"
)

var (
	content = 
`{
    "label": "hello 您好 ~"
}`
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
		{"json escape", content, config.JSONEscape, `{\n    \"label\": \"hello 您好 ~\"\n}`},
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
