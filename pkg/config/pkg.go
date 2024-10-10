package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/provider/file"

	"gopkg.in/yaml.v3"
)

type ChangeFunc func() error

var (
	OnChanged       provider.ChangeFunc
	dynamicProvider provider.Provider
	mainProvider    provider.Provider
)

func Load(path string) (Options, error) {
	// load main config
	var mainOpts Options

	b, err := os.ReadFile(path)
	if err != nil {
		return mainOpts, err
	}

	content := string(b)
	mainOpts, err = unmarshal(content)
	if err != nil {
		return mainOpts, err
	}

	err = ValidateConfig(mainOpts)
	if err != nil {
		var errInvalidConfig ErrInvalidConfig
		if errors.As(err, &errInvalidConfig) {
			line := findConfigurationLine(content, errInvalidConfig.FullPath, errInvalidConfig.Value)
			return mainOpts, fmt.Errorf("%s; in %s:%d", errInvalidConfig.Error(), path, line)
		}
		return mainOpts, err
	}

	dynamicProvider, mainOpts, err = LoadDynamic(mainOpts)
	if err != nil {
		return mainOpts, fmt.Errorf("fail to load dynamic config: %w", err)
	}

	err = ValidateMapping(mainOpts)
	if err != nil {
		return mainOpts, err
	}

	mainOpts.from = path

	fileProviderOpts := file.Options{
		Paths: []string{path},
	}
	mainProvider = file.NewProvider(fileProviderOpts)

	return mainOpts, nil
}

func LoadDynamic(mainOptions Options) (provider.Provider, Options, error) {

	// use file provider if enabled
	if mainOptions.Providers.File.Enabled {

		if len(mainOptions.Providers.File.Paths) == 0 {
			return nil, mainOptions, nil
		}

		fileProviderOpts := file.Options{
			Paths: []string{},
		}
		fileProvider := file.NewProvider(fileProviderOpts)

		for _, content := range mainOptions.Providers.File.Paths {
			fileProvider.Add(content)
		}

		cInfo, err := fileProvider.Open()
		if err != nil {
			return nil, mainOptions, err
		}

		for _, c := range cInfo {
			mainOptions, err = mergeOptions(mainOptions, c.Content)
			if err != nil {
				var errInvalidConfig ErrInvalidConfig
				if errors.As(err, &errInvalidConfig) {
					line := findConfigurationLine(c.Content, errInvalidConfig.FullPath, errInvalidConfig.Value)
					return nil, mainOptions, fmt.Errorf("%s; in %s:%d", errInvalidConfig.Error(), c.Path, line)
				}

				errMsg := fmt.Sprintf("path: %s, error: %s", c.Path, err.Error())
				return nil, mainOptions, errors.New(errMsg)
			}
		}

		return fileProvider, mainOptions, nil
	}

	return nil, mainOptions, nil
}

func Watch() error {
	if mainProvider != nil {
		mainProvider.SetOnChanged(OnChanged)
		_ = mainProvider.Watch()
	}

	if dynamicProvider != nil {
		dynamicProvider.SetOnChanged(OnChanged)
		_ = dynamicProvider.Watch()
	}

	return nil
}

func unmarshal(content string) (Options, error) {
	result := Options{}

	b := []byte(content)

	err := yaml.Unmarshal(b, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func mergeOptions(mainOpts Options, content string) (Options, error) {

	newOptions, err := unmarshal(content)
	if err != nil {
		return mainOpts, err
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

	for k, v := range newOptions.Middlewares {
		if _, found := mainOpts.Middlewares[k]; found {
			msg := fmt.Sprintf("middleware '%s' is duplicate", k)
			fullpath := []string{"middlewares", k}
			return mainOpts, newInvalidConfig(fullpath, "", msg)
		}

		mainOpts.Middlewares[k] = v
	}

	for k, v := range newOptions.Services {
		if _, found := mainOpts.Services[k]; found {
			msg := fmt.Sprintf("service '%s' is duplicate", k)
			fullpath := []string{"services", k}
			return mainOpts, newInvalidConfig(fullpath, "", msg)
		}

		mainOpts.Services[k] = v
	}

	for k, v := range newOptions.Routes {
		if _, found := mainOpts.Routes[k]; found {
			msg := fmt.Sprintf("route '%s' is duplicate", k)
			fullpath := []string{"routes", k}
			return mainOpts, newInvalidConfig(fullpath, "", msg)
		}

		mainOpts.Routes[k] = v
	}

	for k, v := range newOptions.Upstreams {
		if _, found := mainOpts.Upstreams[k]; found {
			msg := fmt.Sprintf("upstream '%s' is duplicate", k)
			fullpath := []string{"upstreams", k}
			return mainOpts, newInvalidConfig(fullpath, "", msg)
		}

		mainOpts.Upstreams[k] = v
	}

	return mainOpts, nil
}

func findConfigurationLine(content string, fullPath []string, value any) int {
	var node any
	var err error

	err = json.Unmarshal([]byte(content), &node)
	if err != nil {
		var yamlNode yaml.Node
		err = yaml.Unmarshal([]byte(content), &yamlNode)
		if err != nil {
			slog.Error("failed to unmarshal config yaml file", "error", err)
			return -1
		}
		node = &yamlNode
	}

	lines := strings.Split(content, "\n")
	return findInNode(node, fullPath, value, lines)
}

func findInNode(node any, path []string, value any, lines []string) int {
	for i, key := range path {
		switch n := node.(type) {
		case map[string]any:
			if val, ok := n[key]; ok {
				if i == len(path)-1 {
					if fmt.Sprintf("%v", val) == fmt.Sprintf("%v", value) {
						return findLineNumber(lines, key, value)
					}
					return -1
				}
				node = val
			} else {
				return -1
			}
		case []any:
			for _, item := range n {
				if line := findInNode(item, path[i:], value, lines); line != -1 {
					return line
				}
			}
			return -1
		case *yaml.Node:
			switch n.Kind {
			case yaml.DocumentNode:
				if len(n.Content) > 0 {
					return findInNode(n.Content[0], path, value, lines)
				}
			case yaml.MappingNode:
				for j := 0; j < len(n.Content); j += 2 {
					if n.Content[j].Value == path[0] {
						if len(path) == 1 {
							if n.Content[j+1].Value == fmt.Sprintf("%v", value) {
								return n.Content[j+1].Line
							}
							return -1
						}
						return findInNode(n.Content[j+1], path[1:], value, lines)
					}
				}
			case yaml.SequenceNode:
				for _, item := range n.Content {
					if line := findInNode(item, path[i:], value, lines); line != -1 {
						return line
					}
				}
			default:
			}
			return -1
		default:
			return -1
		}
	}
	return -1
}

func findLineNumber(lines []string, key string, value any) int {
	searchStr := fmt.Sprintf(`"%s": %v`, key, value)
	for i, line := range lines {
		if strings.Contains(line, searchStr) {
			return i + 1
		}
	}
	return -1
}
