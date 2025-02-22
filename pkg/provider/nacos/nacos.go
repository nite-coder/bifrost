package nacos

import (
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/nite-coder/bifrost/pkg/provider"
)

type Options struct {
	Config Config
}

type ServerConfig struct {
	IPAddr string
	Port   uint64
}

type File struct {
	DataID  string
	Group   string
	Content string
}

type Config struct {
	Namespace   string
	ContextPath string
	LogDir      string
	CacheDir    string
	Timeout     time.Duration
	Watch       *bool
	Servers     []*ServerConfig
	Files       []*File
}

func (c Config) IsWatch() bool {
	if c.Watch == nil {
		return true
	}

	return *c.Watch
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

	logDir := "/tmp/nacos/log"
	if opts.Config.LogDir != "" {
		logDir = opts.Config.LogDir
	}

	cacheDir := "/tmp/nacos/cache"
	if opts.Config.CacheDir != "" {
		cacheDir = opts.Config.CacheDir
	}

	namespace := "public"
	if opts.Config.Namespace != "" {
		namespace = opts.Config.Namespace
	}

	for _, server := range opts.Config.Servers {
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(
			server.IPAddr,
			server.Port,
			constant.WithContextPath(contextPath),
		))
	}

	var timeoutMS uint64
	timeout := opts.Config.Timeout.Milliseconds()
	if timeout <= 0 {
		timeoutMS = 5000
	} else {
		timeoutMS = uint64(timeout)
	}

	configConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(namespace),
		constant.WithTimeoutMs(timeoutMS),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir(logDir),
		constant.WithCacheDir(cacheDir),
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
		content, err := p.client.GetConfig(vo.ConfigParam{
			DataId: file.DataID,
			Group:  file.Group,
		})
		if err != nil {
			return nil, err
		}

		newFile := &File{
			DataID:  file.DataID,
			Group:   file.Group,
			Content: content,
		}

		result = append(result, newFile)
	}

	return result, nil
}

func (p *NacosProvider) Watch() error {
	if !p.options.Config.IsWatch() {
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
