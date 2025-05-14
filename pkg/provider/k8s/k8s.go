package k8s

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sDiscovery struct {
	client kubernetes.Interface
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

func NewK8sDiscovery(cfg *Options) (*K8sDiscovery, error) {
	var config *rest.Config
	var err error

	if cfg == nil {
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
		if cfg.KubeConfig != "" {
			// Use provided kubeconfig file
			config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to load kubeconfig file: %w", err)
			}
		} else {
			// Create config from explicit API server URL
			config = &rest.Config{
				Host: cfg.APIServer,
			}

			if cfg.BearerToken != "" {
				config.BearerToken = cfg.BearerToken
			}

			if cfg.Insecure {
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
		client: clientset,
	}, nil
}

func (k *K8sDiscovery) GetInstances(ctx context.Context, options provider.GetInstanceOptions) ([]provider.Instancer, error) {
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": options.Name,
		},
	}

	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %w", err)
	}

	pods, err := k.client.CoreV1().Pods(options.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	instances := make([]provider.Instancer, 0, len(pods.Items))
	for _, pod := range pods.Items {
		// Check if the pod is ready
		if isPodReady(&pod) {
			instance := k.podToInstance(&pod)
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (k *K8sDiscovery) Watch(ctx context.Context, options provider.GetInstanceOptions) (<-chan []provider.Instancer, error) {
	watcher, err := k.client.CoreV1().Pods(options.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: "app=" + options.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to watch pods: %w", err)
	}

	ch := make(chan []provider.Instancer)

	go safety.Go(ctx, func() {
		defer watcher.Stop()
		defer close(ch)

		for {
			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}

				if !shouldTriggerUpdate(event) {
					continue
				}

				instances, err := k.GetInstances(ctx, options)
				if err != nil {
					continue
				}

				ch <- instances

			case <-ctx.Done():
				return
			}
		}
	})

	return ch, nil
}

func (k *K8sDiscovery) podToInstance(pod *corev1.Pod) provider.Instancer {
	var port int

	// Search through containers for a suitable port
	for _, container := range pod.Spec.Containers {
		// First try to find a named HTTP port
		for _, containerPort := range container.Ports {
			if containerPort.Name == "http" || containerPort.Name == "https" {
				port = int(containerPort.ContainerPort)
				break
			}
		}

		// If no HTTP port found, use the first available port
		if port == 0 && len(container.Ports) > 0 {
			port = int(container.Ports[0].ContainerPort)
			break
		}
	}

	// Fallback to default port if none found
	if port == 0 {
		port = 80
	}

	addr := &net.TCPAddr{
		IP:   net.ParseIP(pod.Status.PodIP),
		Port: port,
	}

	weight := uint32(1)
	if w, exists := pod.Annotations["weight"]; exists {
		if weightVal, err := strconv.ParseUint(w, 10, 32); err == nil {
			weight = uint32(weightVal)
		}
	}

	instance := provider.NewInstance(addr, weight)

	for k, v := range pod.Labels {
		instance.SetTag(k, v)
	}

	// add additional tags
	instance.SetTag("pod-name", pod.Name)
	instance.SetTag("node-name", pod.Spec.NodeName)
	instance.SetTag("namespace", pod.Namespace)

	return instance
}

func isPodReady(pod *corev1.Pod) bool {
	// check pod status
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	// ensure pod ip
	if pod.Status.PodIP == "" {
		return false
	}

	// make sure pod is ready
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func shouldTriggerUpdate(event watch.Event) bool {
	pod, ok := event.Object.(*corev1.Pod)
	if !ok {
		return false
	}

	switch event.Type {
	case watch.Added, watch.Modified:
		return isPodReady(pod)
	case watch.Deleted:
		return true
	default:
		return false
	}
}
