package nacos

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type NacosServiceDiscovery struct {
	client  naming_client.INamingClient
	options *Options
}

func NewNacosServiceDiscovery(options Options) (*NacosServiceDiscovery, error) {
	serverConfigs := []constant.ServerConfig{}

	contextPath := "/nacos"
	if options.Prefix != "" {
		contextPath = options.Prefix
	}

	logDir := ""
	if options.LogDir != "" {
		logDir = options.LogDir
	}

	cacheDir := ""
	if options.CacheDir != "" {
		cacheDir = options.CacheDir
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
			port = uint64(8848)
		}

		serverConfigs = append(serverConfigs, *constant.NewServerConfig(
			ipaddr,
			port,
			constant.WithContextPath(contextPath),
		))
	}

	var timeoutMS uint64
	timeout := options.Timeout.Milliseconds()
	if timeout <= 0 {
		timeoutMS = 10000
	} else {
		timeoutMS = uint64(timeout)
	}

	configConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(options.NamespaceID),
		constant.WithTimeoutMs(timeoutMS),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir(logDir),
		constant.WithCacheDir(cacheDir),
		constant.WithLogLevel(options.LogLevel),
		constant.WithUsername(options.Username),
		constant.WithPassword(options.Password),
	)

	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &configConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, err
	}

	return &NacosServiceDiscovery{
		client:  client,
		options: &options,
	}, nil

}

func (d *NacosServiceDiscovery) GetInstances(ctx context.Context, options provider.GetInstanceOptions) ([]provider.Instancer, error) {
	nacosInstances, err := d.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: options.ID,
		GroupName:   options.Group,
		HealthyOnly: true,
	})

	if err != nil {
		if err.Error() == "instance list is empty!" {
			return nil, nil
		}
		return nil, fmt.Errorf("fail to select instances from nacos, error: %w, discovery id: %s, group: %s", err, options.ID, options.Group)
	}

	instances := ToProviderInstance(nacosInstances)
	return instances, nil
}

func (d *NacosServiceDiscovery) Watch(ctx context.Context, options provider.GetInstanceOptions) (<-chan []provider.Instancer, error) {
	ch := make(chan []provider.Instancer, 1)

	err := d.client.Subscribe(&vo.SubscribeParam{
		ServiceName: options.ID,
		GroupName:   options.Group,
		SubscribeCallback: func(nacosInstances []model.Instance, err error) {
			if err != nil {
				return
			}

			instances := ToProviderInstance(nacosInstances)
			ch <- instances
		},
	})

	if err != nil {
		return nil, err
	}

	return ch, nil
}
