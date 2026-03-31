package trafficsplitter

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// Options defines the configuration for the traffic splitter middleware.
type Options struct {
	Key          string
	Destinations []*Destination
}

// Destination defines a single target and its weight for traffic splitting.
type Destination struct {
	To     string
	Weight int64
}

// Middleware is a middleware that splits traffic among multiple destinations.
type Middleware struct {
	options     *Options
	totalWeight int64
}

// NewMiddleware creates a new TrafficSplitterMiddleware instance.
func NewMiddleware(options *Options) *Middleware {
	m := &Middleware{
		options: options,
	}

	for _, dest := range options.Destinations {
		if dest.Weight <= 0 {
			dest.Weight = 1
		}
		m.totalWeight += dest.Weight
	}

	return m
}

func (m *Middleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if m.totalWeight <= 0 {
		c.Next(ctx)
		return
	}

	randomWeight, err := getRandomNumber(m.totalWeight)
	if err != nil {
		// If random number generation fails, default to the first destination or just proceed
		if len(m.options.Destinations) > 0 {
			c.Set(m.options.Key, m.options.Destinations[0].To)
		}
		c.Next(ctx)
		return
	}

	for _, dest := range m.options.Destinations {
		randomWeight -= dest.Weight
		if randomWeight < 0 {
			c.Set(m.options.Key, dest.To)
			break
		}
	}

	c.Next(ctx)
}
