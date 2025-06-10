package trafficsplitter

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type Options struct {
	Key          string
	Destinations []*Destination
}

type Destination struct {
	To     string
	Weight int64
}

type TrafficSplitterMiddleware struct {
	options     *Options
	totalWeight int64
}

func NewMiddleware(options *Options) *TrafficSplitterMiddleware {

	m := &TrafficSplitterMiddleware{
		options: options,
	}

	for _, dest := range options.Destinations {
		m.totalWeight += dest.Weight
	}

	return m
}

func (m *TrafficSplitterMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	randomWeight, _ := getRandomNumber(m.totalWeight)

	for _, dest := range m.options.Destinations {
		if dest.Weight <= 0 {
			dest.Weight = 1
		}

		randomWeight -= dest.Weight
		if randomWeight < 0 {
			c.Set(m.options.Key, dest.To)
			break
		}
	}

	c.Next(ctx)
}
