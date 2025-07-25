/*
 * The MIT License (MIT)
 *
 * Copyright (c) 2016 Gin-Gonic
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.

 * This file may have been modified by CloudWeGo authors. All CloudWeGo
 * Modifications are Copyright 2022 CloudWeGo Authors.
 */

package cors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

// Config represents all available options for the middleware.
type Config struct {
	// AllowOriginFunc is a custom function to validate the origin. It takes the origin
	// as argument and returns true if allowed or false otherwise.
	// AllowOrigins have a higher AllowOrigins have a higher priority than AllowOriginFunc
	// It is recommended to use AllowOriginFunc without setting AllowOrigins.
	AllowOriginFunc func(origin string) bool
	// AllowOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// Default value is []
	AllowOrigins []string `mapstructure:"allow_origins"`
	// AllowMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (GET and POST)
	AllowMethods []string `mapstructure:"allow_methods"`
	// AllowHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	AllowHeaders []string `mapstructure:"allow_headers"`
	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposeHeaders []string `mapstructure:"expose_headers"`
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge          time.Duration `mapstructure:"max_age"`
	AllowAllOrigins bool          `mapstructure:"allow_all_origins"`
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool `mapstructure:"allow_credentials"`
	// Allows to add origins like http://some-domain/*, https://api.* or http://some.*.subdomain.com
	AllowWildcard bool `mapstructure:"allow_wildcard"`
	// Allows usage of popular browser extensions schemas
	AllowBrowserExtensions bool
	// Allows usage of WebSocket protocol
	AllowWebSockets bool
	// Allows usage of file:// schema (dangerous!) use it only when you 100% sure it's needed
	AllowFiles bool
}

// AddAllowMethods is allowed to add custom methods
func (c *Config) AddAllowMethods(methods ...string) {
	c.AllowMethods = append(c.AllowMethods, methods...)
}

// AddAllowHeaders is allowed to add custom headers
func (c *Config) AddAllowHeaders(headers ...string) {
	c.AllowHeaders = append(c.AllowHeaders, headers...)
}

// AddExposeHeaders is allowed to add custom expose headers
func (c *Config) AddExposeHeaders(headers ...string) {
	c.ExposeHeaders = append(c.ExposeHeaders, headers...)
}
func (c Config) getAllowedSchemas() []string {
	allowedSchemas := DefaultSchemas
	if c.AllowBrowserExtensions {
		allowedSchemas = append(allowedSchemas, ExtensionSchemas...)
	}
	if c.AllowWebSockets {
		allowedSchemas = append(allowedSchemas, WebSocketSchemas...)
	}
	if c.AllowFiles {
		allowedSchemas = append(allowedSchemas, FileSchemas...)
	}
	return allowedSchemas
}
func (c Config) validateAllowedSchemas(origin string) bool {
	allowedSchemas := c.getAllowedSchemas()
	for _, schema := range allowedSchemas {
		if strings.HasPrefix(origin, schema) {
			return true
		}
	}
	return false
}

// Validate is check configuration of user defined.
func (c Config) Validate() error {
	if c.AllowAllOrigins && (c.AllowOriginFunc != nil || len(c.AllowOrigins) > 0) {
		return errors.New("conflict settings: all origins are allowed. AllowOriginFunc or AllowOrigins is not needed")
	}
	if !c.AllowAllOrigins && c.AllowOriginFunc == nil && len(c.AllowOrigins) == 0 {
		return errors.New("conflict settings: all origins disabled")
	}
	for _, origin := range c.AllowOrigins {
		if !strings.Contains(origin, "*") && !c.validateAllowedSchemas(origin) {
			return errors.New("bad origin: origins must contain '*' or include " + strings.Join(c.getAllowedSchemas(), ","))
		}
	}
	return nil
}
func (c Config) parseWildcardRules() [][]string {
	var wRules [][]string
	if !c.AllowWildcard {
		return wRules
	}
	for _, o := range c.AllowOrigins {
		if !strings.Contains(o, "*") {
			continue
		}
		if c := strings.Count(o, "*"); c > 1 {
			panic(errors.New("only one * is allowed").Error())
		}
		i := strings.Index(o, "*")
		if i == 0 {
			wRules = append(wRules, []string{"*", o[1:]})
			continue
		}
		if i == (len(o) - 1) {
			wRules = append(wRules, []string{o[:i-1], "*"})
			continue
		}
		wRules = append(wRules, []string{o[:i], o[i+1:]})
	}
	return wRules
}

// DefaultConfig returns a generic default configuration mapped to localhost.
func DefaultConfig() Config {
	return Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

// Default returns the location middleware with default configuration.
func Default() app.HandlerFunc {
	config := DefaultConfig()
	config.AllowAllOrigins = true
	return NewMiddleware(config)
}

// NewMiddleware returns the location middleware with user-defined custom configuration.
func NewMiddleware(config Config) app.HandlerFunc {
	cors := newCors(config)
	return func(ctx context.Context, c *app.RequestContext) {
		cors.applyCors(c)
	}
}
func init() {
	_ = middleware.Register([]string{"cors"}, func(params any) (app.HandlerFunc, error) {
		cfg := Config{}
		if params == nil {
			cfg = DefaultConfig()
			cfg.AllowAllOrigins = true
		} else {
			decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
				WeaklyTypedInput: true,
				Result:           &cfg,
			})
			err := decoder.Decode(params)
			if err != nil {
				return nil, fmt.Errorf("cors middleware params is invalid: %w", err)
			}
			if len(cfg.AllowMethods) == 0 {
				cfg.AllowMethods = DefaultConfig().AllowMethods
			}
			if len(cfg.AllowHeaders) == 0 {
				cfg.AllowHeaders = DefaultConfig().AllowHeaders
			}
			if cfg.MaxAge == 0 {
				cfg.MaxAge = DefaultConfig().MaxAge
			}
		}
		m := NewMiddleware(cfg)
		return m, nil
	})
}
