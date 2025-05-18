package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
)

func TestGetInstances(t *testing.T) {
	tests := []struct {
		name          string
		endpointSlice *discoveryv1.EndpointSlice
		pods          []corev1.Pod
		options       provider.GetInstanceOptions
		expectedCount int
		wantErr       bool
	}{
		{
			name: "endpointslice with pod ref",
			endpointSlice: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-abc",
					Namespace: "default",
					Labels: map[string]string{
						discoveryv1.LabelServiceName: "test-app",
					},
				},
				AddressType: discoveryv1.AddressTypeIPv4,
				Endpoints: []discoveryv1.Endpoint{
					{
						Addresses: []string{"192.168.1.1"},
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "test-pod-1",
						},
						Conditions: discoveryv1.EndpointConditions{
							Ready: ptr.To(true),
						},
					},
				},
				Ports: []discoveryv1.EndpointPort{
					{
						Port: ptr.To(int32(8080)),
					},
				},
			},
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "default",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 8080,
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						PodIP: "192.168.1.1",
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			options: provider.GetInstanceOptions{
				Name:      "test-app",
				Namespace: "default",
			},
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name: "endpointslice without pod ref",
			endpointSlice: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-abc",
					Namespace: "default",
					Labels: map[string]string{
						discoveryv1.LabelServiceName: "test-app",
					},
				},
				AddressType: discoveryv1.AddressTypeIPv4,
				Endpoints: []discoveryv1.Endpoint{
					{
						Addresses: []string{"192.168.1.1"},
						Conditions: discoveryv1.EndpointConditions{
							Ready: ptr.To(true),
						},
					},
				},
				Ports: []discoveryv1.EndpointPort{
					{
						Port: ptr.To(int32(8080)),
					},
				},
			},
			options: provider.GetInstanceOptions{
				Name:      "test-app",
				Namespace: "default",
			},
			expectedCount: 1,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			client := fake.NewSimpleClientset()

			_, err := client.DiscoveryV1().EndpointSlices(tt.options.Namespace).Create(
				context.Background(),
				tt.endpointSlice,
				metav1.CreateOptions{},
			)
			assert.NoError(t, err)

			// Create test pods if any
			for _, pod := range tt.pods {
				_, err := client.CoreV1().Pods(tt.options.Namespace).Create(
					context.Background(),
					&pod,
					metav1.CreateOptions{},
				)
				assert.NoError(t, err)
			}

			k8sDiscovery := &K8sDiscovery{
				client: client,
			}

			instances, err := k8sDiscovery.GetInstances(context.Background(), tt.options)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, instances, tt.expectedCount)

			// Verify instances are correct
			for _, instance := range instances {
				assert.NotEmpty(t, instance.Address())
			}
		})
	}
}

func TestWatch(t *testing.T) {
	tests := []struct {
		name          string
		endpointSlice *discoveryv1.EndpointSlice
		pods          []corev1.Pod
		options       provider.GetInstanceOptions
		operations    []struct {
			event     watch.EventType
			obj       any
			expected  int
			instance  string
			hasWeight bool
		}
	}{
		{
			name: "endpointslice added",
			endpointSlice: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-abc",
					Namespace: "default",
					Labels: map[string]string{
						discoveryv1.LabelServiceName: "test-app",
					},
				},
				AddressType: discoveryv1.AddressTypeIPv4,
				Endpoints: []discoveryv1.Endpoint{
					{
						Addresses: []string{"192.168.1.1"},
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "test-pod-1",
						},
						Conditions: discoveryv1.EndpointConditions{
							Ready: ptr.To(true),
						},
					},
				},
				Ports: []discoveryv1.EndpointPort{
					{
						Port: ptr.To(int32(8080)),
					},
				},
			},
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "default",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Ports: []corev1.ContainerPort{
									{ContainerPort: 8080},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						PodIP: "192.168.1.1",
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			options: provider.GetInstanceOptions{
				Name:      "test-app",
				Namespace: "default",
			},
			operations: []struct {
				event     watch.EventType
				obj       any
				expected  int
				instance  string
				hasWeight bool
			}{
				{
					event: watch.Added,
					obj: &discoveryv1.EndpointSlice{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-app-xyz",
							Namespace: "default",
							Labels: map[string]string{
								discoveryv1.LabelServiceName: "test-app",
							},
						},
						AddressType: discoveryv1.AddressTypeIPv4,
						Endpoints: []discoveryv1.Endpoint{
							{
								Addresses: []string{"192.168.1.2"},
								TargetRef: &corev1.ObjectReference{
									Kind: "Pod",
									Name: "test-pod-2",
								},
								Conditions: discoveryv1.EndpointConditions{
									Ready: ptr.To(true),
								},
							},
						},
						Ports: []discoveryv1.EndpointPort{
							{
								Port: ptr.To(int32(8080)),
							},
						},
					},
					expected:  2,
					instance:  "192.168.1.2:8080",
					hasWeight: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()

			// Create initial endpointslice and pods
			_, err := client.DiscoveryV1().EndpointSlices(tt.options.Namespace).Create(
				context.Background(),
				tt.endpointSlice,
				metav1.CreateOptions{},
			)
			assert.NoError(t, err)

			for _, pod := range tt.pods {
				_, err := client.CoreV1().Pods(tt.options.Namespace).Create(
					context.Background(),
					&pod,
					metav1.CreateOptions{},
				)
				assert.NoError(t, err)
			}

			k8sDiscovery := &K8sDiscovery{
				client: client,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			ch, err := k8sDiscovery.Watch(ctx, tt.options)
			assert.NoError(t, err)

			for _, op := range tt.operations {
				switch obj := op.obj.(type) {
				case *discoveryv1.EndpointSlice:
					_, err := client.DiscoveryV1().EndpointSlices(tt.options.Namespace).Create(
						context.Background(),
						obj,
						metav1.CreateOptions{},
					)
					assert.NoError(t, err)

					select {
					case instances := <-ch:
						assert.Len(t, instances, op.expected)
						found := false
						for _, instance := range instances {
							t.Logf("Found instance: %s", instance.Address().String())
							if instance.Address().String() == op.instance {
								found = true
								if op.hasWeight {
									assert.Equal(t, uint32(1), instance.Weight())
								}
							}
						}
						assert.True(t, found, "Expected instance %s not found", op.instance)
					case <-time.After(5 * time.Second):
						t.Fatal("timeout waiting for watch update")
					}
				}
			}
		})
	}
}
