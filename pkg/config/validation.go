package config

import (
	"fmt"
	"time"
)

func Validate(mainOpts Options) error {
	if len(mainOpts.Servers) == 0 {
		return fmt.Errorf("no server found")
	}

	if len(mainOpts.Routes) == 0 {
		return fmt.Errorf("no route found")
	}

	for id, opts := range mainOpts.AccessLogs {
		if !opts.Enabled {
			continue
		}

		if opts.Template == "" {
			return fmt.Errorf("access log '%s' template can't be empty", id)
		}

		if len(opts.TimeFormat) > 0 {
			_, err := time.Parse(opts.TimeFormat, time.Now().Format(opts.TimeFormat))
			if err != nil {
				return fmt.Errorf("access log '%s' time format is invalid", id)
			}
		}
	}

	for id, opts := range mainOpts.Servers {
		if opts.Bind == "" {
			return fmt.Errorf("server '%s' bind can't be empty", id)
		}
	}

	for routeID, route := range mainOpts.Routes {
		for _, serverID := range route.Servers {
			if _, found := mainOpts.Servers[serverID]; !found {
				return fmt.Errorf("server '%s' is invalid in '%s' route section", serverID, routeID)
			}
		}
	}

	for upstreamID, opts := range mainOpts.Upstreams {

		if upstreamID[0] == '$' {
			return fmt.Errorf("upstream '%s' is invalid.  name can't start with '$", upstreamID)
		}

		switch opts.Strategy {
		case WeightedStrategy, RandomStrategy, HashingStrategy, RoundRobinStrategy, "":
		default:
			return fmt.Errorf("upstream '%s' strategy field '%s' is invalid", upstreamID, opts.Strategy)
		}

		if opts.Strategy == HashingStrategy && opts.HashOn == "" {
			return fmt.Errorf("upstream '%s' hash_on field can't be empty", upstreamID)
		}
	}

	return nil
}
