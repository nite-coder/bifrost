package gateway

import (
	"fmt"
	"http-benchmark/pkg/domain"
	"os"

	"gopkg.in/yaml.v3"
)

func parseContent(content string) (domain.Options, error) {
	result := domain.Options{}

	b := []byte(content)

	err := yaml.Unmarshal(b, &result)
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

	if mainOpts.Services == nil {
		mainOpts.Services = make(map[string]domain.ServiceOptions)
	}

	for k, v := range otherOpts.Entries {

		if _, found := mainOpts.Entries[k]; found {
			return mainOpts, fmt.Errorf("entry '%s' is duplicate", k)
		}

		mainOpts.Entries[k] = v
	}

	for k, v := range otherOpts.Middlewares {
		if _, found := mainOpts.Middlewares[k]; found {
			return mainOpts, fmt.Errorf("middleware '%s' is duplicate", k)
		}

		mainOpts.Middlewares[k] = v
	}

	for k, v := range otherOpts.Services {
		if _, found := mainOpts.Services[k]; found {
			return mainOpts, fmt.Errorf("service '%s' is duplicate", k)
		}

		mainOpts.Services[k] = v
	}

	for k, v := range otherOpts.Routes {
		if _, found := mainOpts.Routes[k]; found {
			return mainOpts, fmt.Errorf("route '%s' is duplicates", k)
		}

		mainOpts.Routes[k] = v
	}

	for k, v := range otherOpts.Upstreams {
		if _, found := mainOpts.Upstreams[k]; found {
			return mainOpts, fmt.Errorf("upstream '%s' is duplicate", k)
		}

		mainOpts.Upstreams[k] = v
	}

	return mainOpts, nil
}

func fileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}

func validateOptions(opts domain.Options) error {
	if len(opts.Entries) == 0 {
		return fmt.Errorf("no entry found")
	}

	if len(opts.Routes) == 0 {
		return fmt.Errorf("no route found")
	}

	for routeID, route := range opts.Routes {
		for _, entry := range route.Entries {
			if _, found := opts.Entries[entry]; !found {
				return fmt.Errorf("entry '%s' is invalid in '%s' route section", entry, routeID)
			}
		}
	}

	for upstreamID, _ := range opts.Upstreams {

		if upstreamID[0] == '$' {
			return fmt.Errorf("upstream '%s' is invalid.  name can't start with '$", upstreamID)
		}
	}

	return nil
}
