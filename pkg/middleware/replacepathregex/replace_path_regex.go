package replacepathregex

import (
	"context"
	"errors"
	"regexp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

type Config struct {
	Regex       string `mapstructure:"regex"`
	Replacement string `mapstructure:"replacement"`
}

func Init() error {
	return middleware.RegisterTyped([]string{"replace_path_regex"}, func(cfg Config) (app.HandlerFunc, error) {
		if cfg.Regex == "" {
			return nil, errors.New("regex field is not set")
		}
		if cfg.Replacement == "" {
			return nil, errors.New("replacement field is not set")
		}

		m, err := NewMiddleware(cfg.Regex, cfg.Replacement)
		if err != nil {
			return nil, err
		}
		return m.ServeHTTP, nil
	})
}

type ReplacePathRegexMiddleware struct {
	regex       *regexp.Regexp
	replacement []byte
}

func NewMiddleware(regex, replacement string) (*ReplacePathRegexMiddleware, error) {

	if regex == "" || replacement == "" {
		return nil, errors.New("regex or replacement is empty")
	}

	re := regexp.MustCompile(regex)

	return &ReplacePathRegexMiddleware{
		regex:       re,
		replacement: []byte(replacement),
	}, nil
}

func (m *ReplacePathRegexMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	newPath := m.regex.ReplaceAll(c.Request.Path(), m.replacement)
	c.Request.URI().SetPathBytes(newPath)
	c.Next(ctx)
}
