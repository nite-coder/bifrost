package gateway

import (
	"fmt"
	"http-benchmark/pkg/config"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func parseContent(content string) (config.Options, error) {
	result := config.Options{}

	b := []byte(content)

	err := yaml.Unmarshal(b, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func mergeOptions(mainOpts config.Options, content string) (config.Options, error) {

	otherOpts, err := parseContent(content)
	if err != nil {
		return mainOpts, err
	}

	if mainOpts.Entries == nil {
		mainOpts.Entries = make(map[string]config.EntryOptions)
	}

	if mainOpts.Routes == nil {
		mainOpts.Routes = make(map[string]config.RouteOptions)
	}

	if mainOpts.Middlewares == nil {
		mainOpts.Middlewares = make(map[string]config.MiddlwareOptions)
	}

	if mainOpts.Upstreams == nil {
		mainOpts.Upstreams = make(map[string]config.UpstreamOptions)
	}

	if mainOpts.Services == nil {
		mainOpts.Services = make(map[string]config.ServiceOptions)
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

func validateOptions(mainOpts config.Options) error {
	if len(mainOpts.Entries) == 0 {
		return fmt.Errorf("no entry found")
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

	for id, opts := range mainOpts.Entries {
		if opts.Bind == "" {
			return fmt.Errorf("entry '%s' bind can't be empty", id)
		}
	}

	for routeID, route := range mainOpts.Routes {
		for _, entry := range route.Entries {
			if _, found := mainOpts.Entries[entry]; !found {
				return fmt.Errorf("entry '%s' is invalid in '%s' route section", entry, routeID)
			}
		}
	}

	for upstreamID, opts := range mainOpts.Upstreams {

		if upstreamID[0] == '$' {
			return fmt.Errorf("upstream '%s' is invalid.  name can't start with '$", upstreamID)
		}

		switch opts.Strategy {
		case config.WeightedStrategy, config.RandomStrategy, config.HashingStrategy, config.RoundRobinStrategy:
		case "":
			return fmt.Errorf("upstream '%s' strategy field can't be empty", upstreamID)
		default:
			return fmt.Errorf("upstream '%s' strategy field '%s' is invalid", upstreamID, opts.Strategy)
		}

		if opts.Strategy == config.HashingStrategy && opts.HashOn == "" {
			return fmt.Errorf("upstream '%s' hash_on field can't be empty", upstreamID)
		}
	}

	return nil
}
