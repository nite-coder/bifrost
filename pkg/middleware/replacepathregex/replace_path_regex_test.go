package replacepathregex

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestReplacePathRegexMiddleware(t *testing.T) {
	tests := []struct {
		name             string
		regex            string
		replacement      string
		originalPath     string
		originalFullPath string
		expectedPath     string
		expectedFullPath string
		expectedHeader   string
	}{
		{
			name:             "Replace path",
			regex:            "^/api/v1/(.*)$",
			replacement:      "/hoo/$1",
			originalFullPath: "/api/v1/users?name=john",
			expectedFullPath: "/hoo/users?name=john",
		},
		{
			name:             "No replacement needed",
			regex:            "^/api(/v2.*)",
			replacement:      "$1",
			originalFullPath: "/v1/users",
			expectedFullPath: "/v1/users",
		},
		{
			name:             "replace all",
			regex:            "^/(.*)$",
			replacement:      "/app-web/$1",
			originalFullPath: "/apiwww/v1/hello/world/",
			expectedFullPath: "/app-web/apiwww/v1/hello/world/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := middleware.Factory("replace_path_regex")

			params := map[string]any{
				"regex":       tt.regex,
				"replacement": tt.replacement,
			}

			m, err := h(params)
			assert.NoError(t, err)

			ctx := app.NewContext(0)
			ctx.Request.SetRequestURI(tt.originalFullPath)

			m(context.Background(), ctx)

			uri := string(ctx.Request.URI().RequestURI())
			assert.Equal(t, tt.expectedFullPath, uri, "Full Path should be replaced correctly")
		})
	}
}

func TestReplacePathRegexMiddleware_Errors(t *testing.T) {
	h := middleware.Factory("replace_path_regex")

	tests := []struct {
		name        string
		params      any
		expectedErr string
	}{
		{
			name:        "nil params",
			params:      nil,
			expectedErr: "regex field is not set",
		},
		{
			name:        "invalid params type",
			params:      "invalid",
			expectedErr: "failed to decode middleware params",
		},
		{
			name:        "missing regex",
			params:      map[string]any{"replacement": "bar"},
			expectedErr: "regex field is not set",
		},
		{
			name:        "missing replacement",
			params:      map[string]any{"regex": "foo"},
			expectedErr: "replacement field is not set",
		},
		{
			name:        "empty regex",
			params:      map[string]any{"regex": "", "replacement": "bar"},
			expectedErr: "regex field is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h(tt.params)
			if err == nil {
				if tt.expectedErr != "" {
					t.Fatalf("expected error containing %q, got nil", tt.expectedErr)
				}
			} else {
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}
