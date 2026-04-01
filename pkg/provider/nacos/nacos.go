package nacos

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/nite-coder/blackbear/pkg/cast"

	"github.com/nite-coder/bifrost/pkg/provider"
)

// Options defines the configuration for the Nacos configuration provider.
type Options struct {
	Username    string
	Password    string
	NamespaceID string
	Prefix      string
	LogDir      string
	LogLevel    string // debug, info, warn, error
	CacheDir    string
	Endpoints   []string
	Files       []*File
	Timeout     time.Duration
	Watch       bool
}

// File defines the Nacos data ID, group, and content.
type File struct {
	DataID  string
	Group   string
	Content string
}

// Provider implements a configuration provider that reads from Nacos.
type Provider struct {
	client    config_client.IConfigClient
	OnChanged provider.ChangeFunc
	options   Options
}

// NewProvider creates a new NacosProvider instance.
func NewProvider(options Options) (*Provider, error) {
	serverConfigs := []constant.ServerConfig{}
	clientOptions := []constant.ClientOption{
		constant.WithNamespaceId(options.NamespaceID),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogLevel(options.LogLevel),
		constant.WithUsername(options.Username),
		constant.WithPassword(options.Password),
	}
	timeout := options.Timeout.Milliseconds()
	if timeout <= 0 {
		clientOptions = append(clientOptions, constant.WithTimeoutMs(defaultNacosTimeoutMS))
	} else {
		clientOptions = append(clientOptions, constant.WithTimeoutMs(uint64(timeout)))
	}
	configConfig := *constant.NewClientConfig(
		clientOptions...,
	)
	contextPath := "/nacos"
	if options.Prefix != "" {
		contextPath = options.Prefix
	}
	if options.LogDir != "" {
		clientOptions = append(clientOptions, constant.WithLogDir(options.LogDir))
	}
	if options.CacheDir != "" {
		clientOptions = append(clientOptions, constant.WithCacheDir(options.CacheDir))
	}
	for _, endpoint := range options.Endpoints {
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			return nil, fmt.Errorf("invalid nacos endpoint, must start with http:// or https://: %s", endpoint)
		}
		uri, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		ipaddr := uri.Hostname()
		port, _ := cast.ToUint64(uri.Port())
		if port == 0 {
			port = uint64(defaultNacosPort)
		}
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(
			ipaddr,
			port,
			constant.WithContextPath(contextPath),
		))
	}
	configConfig = *constant.NewClientConfig(
		clientOptions...,
	)
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &configConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Provider{
		options: options,
		client:  client,
	}, nil
}

// SetOnChanged sets the callback function to be called when configuration changes in Nacos.
func (p *Provider) SetOnChanged(changeFunc provider.ChangeFunc) {
	p.OnChanged = changeFunc
}

// ConfigOpen reads the configured files from Nacos and returns their content.
func (p *Provider) ConfigOpen() ([]*File, error) {
	result := make([]*File, 0, len(p.options.Files))
	for _, file := range p.options.Files {
		group := "DEFAULT_GROUP"
		if file.Group != "" {
			group = file.Group
		}
		content, err := p.client.GetConfig(vo.ConfigParam{
			DataId: file.DataID,
			Group:  file.Group,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"nacos: failed to get '%s' file from '%s' group in '%s' namespace_id, error: %w",
				file.DataID,
				file.Group,
				p.options.NamespaceID,
				err,
			)
		}
		newFile := &File{
			DataID:  file.DataID,
			Group:   group,
			Content: content,
		}
		result = append(result, newFile)
	}
	return result, nil
}

// Watch starts listening for configuration changes in Nacos.
func (p *Provider) Watch() error {
	if !p.options.Watch {
		return nil
	}
	for _, file := range p.options.Files {
		err := p.client.ListenConfig(vo.ConfigParam{
			DataId: file.DataID,
			Group:  file.Group,
			OnChange: func(_, _, _, _ string) {
				_ = p.OnChanged()
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
