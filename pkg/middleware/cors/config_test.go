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
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/route"
)

func TestDefault(t *testing.T) {
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(Default())
	router.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(http.StatusOK, "get")
	})
	w := performRequest(router, "GET", "http://facebook.com")
	assert.DeepEqual(t, "get", w.Body.String())
	assert.DeepEqual(t, 200, w.Code)
	assert.DeepEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Expose-Headers"))
}

func TestConfig_parseWildcardRules(t *testing.T) {
	assert.Panic(t, func() {
		c := DefaultConfig()
		c.AllowWildcard = true
		c.AllowOrigins = []string{
			"www.*.*",
		}
		c.parseWildcardRules()
	})
}
