package k8s

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/provider"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sDiscovery struct {
	options *Options
	client  kubernetes.Interface
}

type Options struct {
	// APIServer is the Kubernetes API server URL (e.g., "https://kubernetes.default.svc")
	APIServer string
	// KubeConfig is the path to kubeconfig file
	KubeConfig string
	// BearerToken is the authentication token
	BearerToken string
	// Insecure allows insecure server connections when using HTTPS
	Insecure bool
}

func NewK8sDiscovery(options Options) (*K8sDiscovery, error) {
	var config *rest.Config
	var err error

	if options.APIServer == "" {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fallback to default kubeconfig
			config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
			if err != nil {
				return nil, fmt.Errorf("failed to get k8s config: %w", err)
			}
		}
	} else {
		if options.KubeConfig != "" {
			// Use provided kubeconfig file
			config, err = clientcmd.BuildConfigFromFlags("", options.KubeConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to load kubeconfig file: %w", err)
			}
		} else {
			// Create config from explicit API server URL
			config = &rest.Config{
				Host: options.APIServer,
			}

			if options.BearerToken != "" {
				config.BearerToken = options.BearerToken
			}

			if options.Insecure {
				config.TLSClientConfig = rest.TLSClientConfig{
					Insecure: true,
				}
			}
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &K8sDiscovery{
		options: &options,
		client:  clientset,
	}, nil
}

func (k *K8sDiscovery) GetInstances(ctx context.Context, options provider.GetInstanceOptions) ([]provider.Instancer, error) {
	logger := log.FromContext(ctx)

	if options.Name == "" {
		return nil, errors.New("service name is required for k8s provider")
	}

	if options.Namespace == "" {
		options.Namespace = "default"
	}

	instances := make([]provider.Instancer, 0)

	slices, err := k.client.DiscoveryV1().EndpointSlices(options.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", options.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint slices: %w", err)
	}

	for _, slice := range slices.Items {
		for _, endpoint := range slice.Endpoints {

			if endpoint.Conditions.Ready != nil && !*endpoint.Conditions.Ready {
				continue
			}

			for _, address := range endpoint.Addresses {
				for _, port := range slice.Ports {
					if port.Port == nil {
						continue
					}

					addr := &net.TCPAddr{
						IP:   net.ParseIP(address),
						Port: int(*port.Port),
					}

					instance := provider.NewInstance(addr, 1)

					if endpoint.TargetRef != nil && endpoint.TargetRef.Kind == "Pod" {
						instance.SetTag("pod-name", endpoint.TargetRef.Name)
						instance.SetTag("namespace", options.Namespace)
					}

					instances = append(instances, instance)
				}
			}
		}
	}

	logger.Info("discovered instances", "count", len(instances))
	return instances, nil
}

func (k *K8sDiscovery) Watch(ctx context.Context, options provider.GetInstanceOptions) (<-chan []provider.Instancer, error) {
	logger := log.FromContext(ctx)

	endpointsWatcher, err := k.client.DiscoveryV1().EndpointSlices(options.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", options.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to watch endpoint slices: %w", err)
	}

	ch := make(chan []provider.Instancer)

	go safety.Go(ctx, func() {
		defer endpointsWatcher.Stop()
		defer close(ch)

		for {
			select {
			case event, ok := <-endpointsWatcher.ResultChan():
				if !ok {
					return
				}

				switch event.Type {
				case watch.Added, watch.Modified:
					endpointSlice, ok := event.Object.(*discoveryv1.EndpointSlice)
					if !ok {
						continue
					}

					if isEndpointSliceReady(endpointSlice) {
						instances, err := k.GetInstances(ctx, options)
						if err != nil {
							logger.Warn("failed to get instances after endpoints update", "error", err.Error())
							continue
						}
						ch <- instances
					}
				case watch.Deleted:
					ch <- []provider.Instancer{} // service is down
				case watch.Bookmark:
					// Skip bookmark events
					continue
				case watch.Error:
					logger.Warn("received error event from endpoints watcher")
					continue
				}
			case <-ctx.Done():
				return
			}
		}
	})

	return ch, nil
}

func isEndpointSliceReady(endpointSlice *discoveryv1.EndpointSlice) bool {
	if len(endpointSlice.Endpoints) == 0 {
		return false
	}

	for _, endpoint := range endpointSlice.Endpoints {
		if endpoint.Conditions.Ready != nil && !*endpoint.Conditions.Ready {
			continue
		}
		if len(endpoint.Addresses) == 0 {
			return false
		}
	}

	return len(endpointSlice.Ports) != 0
}
