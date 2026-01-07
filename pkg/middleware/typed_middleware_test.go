package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Prefix     string `mapstructure:"prefix"`
	MaxRetries int    `mapstructure:"max_retries"`
}

func TestRegisterTyped(t *testing.T) {
	// register a typed middleware
	err := RegisterTyped([]string{"typed_test_middleware"}, func(cfg TestConfig) (app.HandlerFunc, error) {
		return func(ctx context.Context, c *app.RequestContext) {
			c.Set("prefix", cfg.Prefix)
			c.Set("retries", cfg.MaxRetries)
		}, nil
	})
	assert.NoError(t, err)

	// simulate gateway factory usage
	factory := Factory("typed_test_middleware")
	assert.NotNil(t, factory)

	// test with valid params (as if coming from yaml/map[string]any)
	params := map[string]any{
		"prefix":      "/api",
		"max_retries": 3,
	}

	handler, err := factory(params)
	assert.NoError(t, err)

	// execute handler
	ctx := context.Background()
	c := app.NewContext(0)
	handler(ctx, c)

	// verify values
	assert.Equal(t, "/api", c.GetString("prefix"))
	assert.Equal(t, 3, c.GetInt("retries"))
}

func TestRegisterTyped_DefaultValues(t *testing.T) {
	// register another typed middleware
	err := RegisterTyped([]string{"typed_test_default"}, func(cfg TestConfig) (app.HandlerFunc, error) {
		return func(ctx context.Context, c *app.RequestContext) {
			c.Set("prefix", cfg.Prefix)
			c.Set("retries", cfg.MaxRetries)
		}, nil
	})
	assert.NoError(t, err)

	factory := Factory("typed_test_default")

	// test with empty params
	handler, err := factory(nil)
	assert.NoError(t, err)

	c := app.NewContext(0)
	handler(context.Background(), c)

	assert.Equal(t, "", c.GetString("prefix"))
	assert.Equal(t, 0, c.GetInt("retries"))
}
