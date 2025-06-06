package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/provider/file"
	"github.com/nite-coder/bifrost/pkg/provider/nacos"
	"github.com/nite-coder/bifrost/pkg/resolver"

	"gopkg.in/yaml.v3"
)

type ChangeFunc func() error

var (
	OnChanged        provider.ChangeFunc
	dynamicProviders []provider.Provider
	mainProvider     provider.Provider
	domainRegex      = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	dnsResolver      *resolver.Resolver
)

func TestAndSkipResovler(path string) (string, error) {
	mainOptions, err := load(path, true)
	return mainOptions.configPath, err
}

func Load(path string) (Options, error) {
	return load(path, false)
}

func load(path string, skipResolver bool) (Options, error) {
	// load main config
	var mainOpts Options

	path, err := defaultPath(path)
	if err != nil {
		return mainOpts, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return mainOpts, err
	}

	content := string(b)
	mainOpts, err = unmarshal(content)
	if err != nil {
		return mainOpts, err
	}

	err = ValidateConfig(mainOpts, false)
	if err != nil {
		var errInvalidConfig ErrInvalidConfig
		if errors.As(err, &errInvalidConfig) {
			line := findConfigurationLine(content, errInvalidConfig.Structure, errInvalidConfig.Value)
			return mainOpts, fmt.Errorf("%s; in %s:%d", errInvalidConfig.Error(), path, line)
		}
		return mainOpts, err
	}

	dps, mainOpts, err := loadDynamic(mainOpts)
	if err != nil {
		return mainOpts, fmt.Errorf("failed to load dynamic config: %w", err)
	}

	if skipResolver {
		mainOpts.SkipResolver = true
	}

	err = ValidateConfig(mainOpts, true)
	if err != nil {
		return mainOpts, err
	}

	mainOpts.configPath = path

	fileProviderOpts := file.Options{
		Paths: []string{path},
	}

	mainProvider = file.NewProvider(fileProviderOpts)
	dynamicProviders = dps

	return mainOpts, nil
}

func loadDynamic(mainOptions Options) ([]provider.Provider, Options, error) {

	providers := make([]provider.Provider, 0)

	// file provider
	if mainOptions.Providers.File.Enabled {
		if len(mainOptions.Providers.File.Paths) == 0 {
			return nil, mainOptions, nil
		}

		fileOptions := file.Options{
			Paths:      []string{},
			Extensions: mainOptions.Providers.File.Extensions,
		}

		if mainOptions.IsWatch() {
			fileOptions.Watch = true
		}

		fileProvider := file.NewProvider(fileOptions)

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
					line := findConfigurationLine(c.Content, errInvalidConfig.Structure, errInvalidConfig.Value)
					return nil, mainOptions, fmt.Errorf("%s; in %s:%d", errInvalidConfig.Error(), c.Path, line)
				}

				errMsg := fmt.Sprintf("path: %s, error: %s", c.Path, err.Error())
				return nil, mainOptions, errors.New(errMsg)
			}
		}

		providers = append(providers, fileProvider)
	}

	// nacos provider
	if mainOptions.Providers.Nacos.Config.Enabled {

		nacosConfigOptions := nacos.Options{
			NamespaceID: mainOptions.Providers.Nacos.Config.NamespaceID,
			Username:    mainOptions.Providers.Nacos.Config.Username,
			Password:    mainOptions.Providers.Nacos.Config.Password,
			Prefix:      mainOptions.Providers.Nacos.Config.Prefix,
			LogDir:      mainOptions.Providers.Nacos.Config.LogDir,
			LogLevel:    mainOptions.Providers.Nacos.Config.LogLevel,
			CacheDir:    mainOptions.Providers.Nacos.Config.CacheDir,
			Timeout:     mainOptions.Providers.Nacos.Config.Timeout,
			Endpoints:   make([]string, 0),
			Files:       make([]*nacos.File, 0),
		}

		if mainOptions.IsWatch() {
			nacosConfigOptions.Watch = true
		}

		nacosConfigOptions.Endpoints = append(nacosConfigOptions.Endpoints, mainOptions.Providers.Nacos.Config.Endpoints...)

		for _, file := range mainOptions.Providers.Nacos.Config.Files {
			nacosConfigOptions.Files = append(nacosConfigOptions.Files, &nacos.File{
				DataID: file.DataID,
				Group:  file.Group,
			})
		}

		nacosProvider, err := nacos.NewProvider(nacosConfigOptions)
		if err != nil {
			return nil, mainOptions, err
		}

		files, err := nacosProvider.ConfigOpen()
		if err != nil {
			return nil, mainOptions, err
		}

		for _, file := range files {
			mainOptions, err = mergeOptions(mainOptions, file.Content)
			if err != nil {
				var errInvalidConfig ErrInvalidConfig
				if errors.As(err, &errInvalidConfig) {
					line := findConfigurationLine(file.Content, errInvalidConfig.Structure, errInvalidConfig.Value)
					return nil, mainOptions, fmt.Errorf("%s; in %s:%d", errInvalidConfig.Error(), file.DataID, line)
				}

				errMsg := fmt.Sprintf("data_id: %s, error: %s", file.DataID, err.Error())
				return nil, mainOptions, errors.New(errMsg)
			}
		}

		providers = append(providers, nacosProvider)
	}

	return providers, mainOptions, nil
}

func Watch() error {
	if mainProvider != nil {
		mainProvider.SetOnChanged(OnChanged)
		_ = mainProvider.Watch()
	}

	if len(dynamicProviders) > 0 {
		for _, dynamicProvider := range dynamicProviders {
			dynamicProvider.SetOnChanged(OnChanged)
			_ = dynamicProvider.Watch()
		}
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

	if mainOpts.Middlewares == nil {
		mainOpts.Middlewares = make(map[string]MiddlwareOptions)
	}

	if mainOpts.Servers == nil {
		mainOpts.Servers = make(map[string]ServerOptions)
	}

	if mainOpts.Routes == nil {
		mainOpts.Routes = make([]*RouteOptions, 0)
	}

	if mainOpts.Services == nil {
		mainOpts.Services = make(map[string]ServiceOptions)
	}

	if mainOpts.Upstreams == nil {
		mainOpts.Upstreams = make(map[string]UpstreamOptions)
	}

	for k, v := range newOptions.Middlewares {
		if _, found := mainOpts.Middlewares[k]; found {
			msg := fmt.Sprintf("middleware '%s' is duplicated", k)
			structure := []string{"middlewares", k}
			return mainOpts, newInvalidConfig(structure, "", msg)
		}

		mainOpts.Middlewares[k] = v
	}

	for k, v := range newOptions.Servers {
		if _, found := mainOpts.Servers[k]; found {
			msg := fmt.Sprintf("server '%s' is duplicated", k)
			structure := []string{"servers", k}
			return mainOpts, newInvalidConfig(structure, "", msg)
		}

		mainOpts.Servers[k] = v
	}

	for _, route := range newOptions.Routes {
		for _, mainRoute := range mainOpts.Routes {
			if mainRoute.ID == route.ID {
				msg := fmt.Sprintf("route '%s' is duplicated", route.ID)
				structure := []string{"routes", route.ID}
				return mainOpts, newInvalidConfig(structure, "", msg)
			}
		}

		mainOpts.Routes = append(mainOpts.Routes, route)
	}

	for k, v := range newOptions.Services {
		if _, found := mainOpts.Services[k]; found {
			msg := fmt.Sprintf("service '%s' is duplicated", k)
			structure := []string{"services", k}
			return mainOpts, newInvalidConfig(structure, "", msg)
		}

		mainOpts.Services[k] = v
	}

	for k, v := range newOptions.Upstreams {
		if _, found := mainOpts.Upstreams[k]; found {
			msg := fmt.Sprintf("upstream '%s' is duplicated", k)
			structure := []string{"upstreams", k}
			return mainOpts, newInvalidConfig(structure, "", msg)
		}

		mainOpts.Upstreams[k] = v
	}

	return mainOpts, nil
}

func findConfigurationLine(content string, fullPath []string, value any) int {
	var node any
	var err error

	err = sonic.Unmarshal([]byte(content), &node)
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

func IsValidDomain(domain string) bool {
	// Define the regular expression
	// Each label: 1-63 characters, can only contain letters, numbers, and hyphens, cannot start or end with a hyphen
	// Full domain: Multiple labels separated by dots, total length should not exceed 253 characters

	if domainRegex.MatchString(domain) && len(domain) <= 253 {
		return true
	}
	return false
}

func defaultPath(path string) (string, error) {
	if path == "" {
		defaultPaths := []string{
			"./config.yaml",
			"./conf/config.yaml",
			"./config/config.yaml",
		}

		for _, dpath := range defaultPaths {
			_, err := os.Stat(dpath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
			}

			path = dpath
			break
		}

		if path == "" {
			return "", errors.New("config file not found")
		}
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path, errors.New("config file not found")
	}

	return path, nil
}
