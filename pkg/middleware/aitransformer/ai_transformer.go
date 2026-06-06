package aitransformer

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/middleware"
)

// Options defines the configuration for the ai_transformer middleware.
type Options struct {
	Format string `mapstructure:"format"` // e.g., "openai-chat", "anthropic", "gemini"
}

// Init registers the ai_transformer middleware to the global middleware factory.
func Init() error {
	return middleware.RegisterTyped([]string{"ai_transformer"}, func(opts Options) (app.HandlerFunc, error) {
		if len(opts.Format) == 0 {
			return nil, errors.New("format parameter is missing or invalid")
		}

		m := NewMiddleware(opts)
		return m.ServeHTTP, nil
	})
}

// Middleware handles the Ingress/Egress translation for AI requests.
type Middleware struct {
	format string
}

// NewMiddleware creates a new ai_transformer middleware instance.
func NewMiddleware(opts Options) *Middleware {
	return &Middleware{
		format: opts.Format,
	}
}

// ServeHTTP implements the core logic for translating AI requests and intercepting errors.
func (m *Middleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	// --- Phase 1: Ingress (Before Next) ---

	// 1. Get ClientAdapter and put it in context for AIProxy to use
	// adapter, _ := ai.GetClientAdapter(m.format)
	// c.Set(ai.ContextKeyClientAdapter, adapter)

	// 2. Translate raw client body to canonical Request object
	// switch family {
	// case ai.FamilyChat:
	//     req, _ := adapter.ToChatRequest(c.Request.Body())
	//     c.Set(ai.ContextKeyChatRequest, req)
	//     
	//     // 🚨 FIX 1.2: Preserve original model name to avoid mutation side-effects
	//     c.Set(ai.ContextKeyVirtualModelName, req.Model)
	//     
	//     // Inject dynamic routing variable with namespace for Service to consume
	//     // c.Set(variable.AIModelName, "ai:" + req.Model)
	// ...
	// }

	c.Next(ctx)
	// ... (rest unchanged)
}
