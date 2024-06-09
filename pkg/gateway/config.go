package gateway

import (
	"fmt"
	"http-benchmark/pkg/domain"

	"gopkg.in/yaml.v3"
)

func parseContent(content string) (domain.Options, error) {
	result := domain.Options{}

	err := yaml.Unmarshal([]byte(content), &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func mergedOptions(mainOpts domain.Options, content string) (domain.Options, error) {

	otherOpts, err := parseContent(content)
	if err != nil {
		return mainOpts, err
	}

	if mainOpts.Entries == nil {
		mainOpts.Entries = make(map[string]domain.EntryOptions)
	}

	if mainOpts.Routes == nil {
		mainOpts.Routes = make(map[string]domain.RouteOptions)
	}

	if mainOpts.Middlewares == nil {
		mainOpts.Middlewares = make(map[string]domain.MiddlwareOptions)
	}

	if mainOpts.Upstreams == nil {
		mainOpts.Upstreams = make(map[string]domain.UpstreamOptions)
	}

	if mainOpts.Transports == nil {
		mainOpts.Transports = make(map[string]domain.TransportOptions)
	}

	for k, v := range otherOpts.Entries {

		if _, found := mainOpts.Entries[k]; found {
			return mainOpts, fmt.Errorf("entry '%s' already exists", k)
		}

		mainOpts.Entries[k] = v
	}

	for k, v := range otherOpts.Routes {
		if _, found := mainOpts.Routes[k]; found {
			return mainOpts, fmt.Errorf("route '%s' already exists", k)
		}

		mainOpts.Routes[k] = v
	}

	for k, v := range otherOpts.Middlewares {
		if _, found := mainOpts.Middlewares[k]; found {
			return mainOpts, fmt.Errorf("middleware '%s' already exists", k)
		}

		mainOpts.Middlewares[k] = v
	}

	for k, v := range otherOpts.Upstreams {
		if _, found := mainOpts.Upstreams[k]; found {
			return mainOpts, fmt.Errorf("upstream '%s' already exists", k)
		}

		mainOpts.Upstreams[k] = v
	}

	for k, v := range otherOpts.Transports {
		if _, found := mainOpts.Transports[k]; found {
			return mainOpts, fmt.Errorf("transport '%s' already exists", k)
		}

		mainOpts.Transports[k] = v
	}

	return mainOpts, nil
}
