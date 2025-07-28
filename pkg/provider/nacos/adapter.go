package nacos

import (
	"net"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/blackbear/pkg/cast"
)

func ToProviderInstance(nacosInstances []model.Instance) []provider.Instancer {
	instances := make([]provider.Instancer, 0)
	for _, nacosInstance := range nacosInstances {
		weight, _ := cast.ToUint32(nacosInstance.Weight)
		if weight == 0 {
			weight = 1
		}

		ip := nacosInstance.Ip
		if nacosInstance.Port > 0 {
			port, _ := cast.ToString(nacosInstance.Port)
			ip = net.JoinHostPort(ip, port)
		} else {
			ip = net.JoinHostPort(ip, "0")
		}

		addr, err := net.ResolveTCPAddr("tcp", ip)
		if err != nil {
			continue
		}

		instance := provider.NewInstance(addr, weight)

		if len(nacosInstance.Metadata) > 0 {
			for key, val := range nacosInstance.Metadata {
				key = strings.TrimSpace(key)
				val = strings.TrimSpace(val)
				instance.SetTag(key, val)
			}
		}

		instances = append(instances, instance)
	}

	return instances
}
