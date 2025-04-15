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
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type Options struct {
	Config Config
}

type File struct {
	DataID  string
	Group   string
	Content string
}

type Config struct {
	Username    string
	Password    string
	NamespaceID string
	ContextPath string
	LogDir      string
	LogLevel    string // debug, info, warn, error
	CacheDir    string
	Timeout     time.Duration
	Watch       bool
	Endpoints   []string
	Files       []*File
}

type NacosProvider struct {
	client    config_client.IConfigClient
	options   Options
	OnChanged provider.ChangeFunc
}

func NewProvider(opts Options) (*NacosProvider, error) {
	serverConfigs := []constant.ServerConfig{}

	contextPath := "/nacos"
	if opts.Config.ContextPath != "" {
		contextPath = opts.Config.ContextPath
	}

	logDir := "./logs"
	if opts.Config.LogDir != "" {
		logDir = opts.Config.LogDir
	}

	cacheDir := "./logs/nacos/cache"
	if opts.Config.CacheDir != "" {
		cacheDir = opts.Config.CacheDir
	}

	for _, endpoint := range opts.Config.Endpoints {
		if (!strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://")) {
			return nil, fmt.Errorf("invalid nacos endpoint, must start with http:// or https://: %s", endpoint)
		}

		uri, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}

		ipaddr := uri.Hostname()
		port, _ := cast.ToUint64(uri.Port())

		if port == 0 {
			port = uint64(8848)
		}

		serverConfigs = append(serverConfigs, *constant.NewServerConfig(
			ipaddr,
			port,
			constant.WithContextPath(contextPath),
		))
	}

	var timeoutMS uint64
	timeout := opts.Config.Timeout.Milliseconds()
	if timeout <= 0 {
		timeoutMS = 10000
	} else {
		timeoutMS = uint64(timeout)
	}

	configConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(opts.Config.NamespaceID),
		constant.WithTimeoutMs(timeoutMS),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir(logDir),
		constant.WithCacheDir(cacheDir),
		constant.WithLogLevel(opts.Config.LogLevel),
		constant.WithUsername(opts.Config.Username),
		constant.WithPassword(opts.Config.Password),
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

	return &NacosProvider{
		options: opts,
		client:  client,
	}, nil
}

func (p *NacosProvider) SetOnChanged(changeFunc provider.ChangeFunc) {
	p.OnChanged = changeFunc
}

func (p *NacosProvider) ConfigOpen() ([]*File, error) {
	result := make([]*File, 0, len(p.options.Config.Files))

	for _, file := range p.options.Config.Files {

		group := "DEFAULT_GROUP"
		if file.Group != "" {
			group = file.Group
		}

		content, err := p.client.GetConfig(vo.ConfigParam{
			DataId: file.DataID,
			Group:  file.Group,
		})
		if err != nil {
			return nil, fmt.Errorf("nacos: fail to get '%s' file from '%s' group in '%s' namespace_id, error: %w", file.DataID, file.Group, p.options.Config.NamespaceID, err)
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

func (p *NacosProvider) Watch() error {
	if !p.options.Config.Watch {
		return nil
	}

	for _, file := range p.options.Config.Files {
		err := p.client.ListenConfig(vo.ConfigParam{
			DataId: file.DataID,
			Group:  file.Group,
			OnChange: func(namespace, group, dataId, data string) {
				_ = p.OnChanged()
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}
