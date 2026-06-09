package aitransformer

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/ai"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
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

	adapter, err := ai.GetClientAdapter(m.format)
	if err != nil {
		c.SetStatusCode(http.StatusInternalServerError)
		c.Abort()
		return
	}
	c.Set(ai.ContextKeyClientAdapter, adapter)

	path := string(c.Request.Path())
	family := ai.FamilyChat
	if strings.HasSuffix(path, "/responses") {
		family = ai.FamilyResponses
	}

	switch family {
	case ai.FamilyChat:
		chatReq, err := adapter.ToChatRequest(c.Request.Body())
		if err != nil {
			abortWithAIError(c, adapter, err)
			return
		}
		c.Set(ai.ContextKeyChatRequest, chatReq)
		c.Set(ai.ContextKeyVirtualModelName, chatReq.Model)
		c.Set(variable.Model, chatReq.Model)
		c.Set(ai.ContextKeyAIFamily, ai.FamilyChat)
	case ai.FamilyResponses:
		respReq, err := adapter.ToResponsesRequest(c.Request.Body())
		if err != nil {
			abortWithAIError(c, adapter, err)
			return
		}
		c.Set(ai.ContextKeyResponsesRequest, respReq)
		c.Set(ai.ContextKeyVirtualModelName, respReq.Model)
		c.Set(variable.Model, respReq.Model)
		c.Set(ai.ContextKeyAIFamily, ai.FamilyResponses)
	default:
	}

	c.Next(ctx)

	// --- Phase 2: Egress (After Next) ---
	if len(c.Errors) > 0 {
		var aiErr *ai.AIError
		for i := len(c.Errors) - 1; i >= 0; i-- {
			errObj := c.Errors[i].Err
			if errors.As(errObj, &aiErr) {
				c.Set(variable.ErrorType, aiErr.Type)
				c.Set(variable.ErrorMessage, aiErr.Message)

				formattedErr, translateErr := adapter.ToClientError(aiErr)
				if translateErr == nil {
					c.JSON(aiErr.StatusCode, formattedErr)
					c.Abort()
					return
				}
			}
		}

		// SECURITY: Handle plain errors (from upstream non-standard responses)
		// Log full details internally, return generic error to client
		for i := len(c.Errors) - 1; i >= 0; i-- {
			errObj := c.Errors[i].Err
			var aiErr *ai.AIError
			if !errors.As(errObj, &aiErr) {
				// This is a plain error (not AIError)
				routeID := variable.GetString(variable.RouteID, c)
				virtualModel := ""
				if vm, ok := c.Get(ai.ContextKeyVirtualModelName); ok {
					if vmStr, ok := vm.(string); ok {
						virtualModel = vmStr
					}
				}

				slog.ErrorContext(ctx, "upstream error intercepted",
					"route_id", routeID,
					"virtual_model", virtualModel,
					"error", errObj.Error(),
				)

				genericErr := &ai.AIError{
					Type:       "internal_error",
					Message:    "Internal server error",
					StatusCode: http.StatusBadGateway,
				}
				c.Set(variable.ErrorType, genericErr.Type)
				c.Set(variable.ErrorMessage, errObj.Error())
				formattedErr, _ := adapter.ToClientError(genericErr)
				c.JSON(http.StatusBadGateway, formattedErr)
				c.Abort()
				return
			}
		}
	}
}

func abortWithAIError(c *app.RequestContext, adapter ai.ClientAdapter, err error) {
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		aiErr = &ai.AIError{
			Type:       "invalid_request_error",
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
		}
	}
	c.Set(variable.ErrorType, aiErr.Type)
	c.Set(variable.ErrorMessage, aiErr.Message)
	formattedErr, _ := adapter.ToClientError(aiErr)
	c.JSON(aiErr.StatusCode, formattedErr)
	c.Abort()
}
