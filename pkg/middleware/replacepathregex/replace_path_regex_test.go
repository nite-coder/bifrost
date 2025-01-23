package replacepathregex

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestReplacePathRegexMiddleware_ServeHTTP(t *testing.T) {
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
			middleware := NewMiddleware(tt.regex, tt.replacement)

			ctx := app.NewContext(0)
			ctx.Request.SetRequestURI(tt.originalFullPath)

			middleware.ServeHTTP(context.Background(), ctx)

			uri := string(ctx.Request.URI().RequestURI())
			assert.Equal(t, tt.expectedFullPath, uri, "Full Path should be replaced correctly")
		})
	}
}
