package config

import (
	"fmt"
	"http-benchmark/pkg/provider/file"

	"gopkg.in/yaml.v3"
)

type ChangeFunc func() error

var (
	OnChanged    ChangeFunc
	providerType string
)

func ProviderType() string {
	return providerType
}

func LoadFrom(path string) (Options, error) {
	// load main config
	var mainOpts Options

	fileProviderOpts := file.Options{
		Paths: []string{path},
	}

	fileProvider := file.NewProvider(fileProviderOpts)

	cInfo, err := fileProvider.Open()
	if err != nil {
		return mainOpts, err
	}

	mainOpts, err = parseContent(cInfo[0].Content)
	if err != nil {
		return mainOpts, err
	}
	fileProvider.Reset()

	// use file provider if enabled
	if mainOpts.Providers.File.Enabled && len(mainOpts.Providers.File.Paths) > 0 {
		for _, content := range mainOpts.Providers.File.Paths {
			fileProvider.Add(content)
		}

		cInfo, err = fileProvider.Open()
		if err != nil {
			return mainOpts, err
		}

		for _, c := range cInfo {
			mainOpts, err = mergeOptions(mainOpts, c.Content)
			if err != nil {
				errMsg := fmt.Sprintf("path: %s, error: %s", c.Path, err.Error())
				return mainOpts, fmt.Errorf(errMsg)
			}
		}

		providerType = "file"
		return mainOpts, nil
	}

	return mainOpts, nil
}

func parseContent(content string) (Options, error) {
	result := Options{}

	b := []byte(content)

	err := yaml.Unmarshal(b, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func mergeOptions(mainOpts Options, content string) (Options, error) {

	otherOpts, err := parseContent(content)
	if err != nil {
		return mainOpts, err
	}

	if mainOpts.Servers == nil {
		mainOpts.Servers = make(map[string]ServerOptions)
	}

	if mainOpts.Routes == nil {
		mainOpts.Routes = make(map[string]RouteOptions)
	}

	if mainOpts.Middlewares == nil {
		mainOpts.Middlewares = make(map[string]MiddlwareOptions)
	}

	if mainOpts.Upstreams == nil {
		mainOpts.Upstreams = make(map[string]UpstreamOptions)
	}

	if mainOpts.Services == nil {
		mainOpts.Services = make(map[string]ServiceOptions)
	}

	for k, v := range otherOpts.Servers {

		if _, found := mainOpts.Servers[k]; found {
			return mainOpts, fmt.Errorf("server '%s' is duplicate", k)
		}

		mainOpts.Servers[k] = v
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
