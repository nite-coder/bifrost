package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetInstances(t *testing.T) {
	tests := []struct {
		name          string
		pods          []corev1.Pod
		options       provider.GetInstanceOptions
		expectedCount int
		wantErr       bool
	}{
		{
			name: "no pods",
			pods: []corev1.Pod{},
			options: provider.GetInstanceOptions{
				Name:      "test-app",
				Namespace: "default",
			},
			expectedCount: 0,
			wantErr:       false,
		},
		{
			name: "one ready pod",
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
										Name:          "http",
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
			name: "one not ready pod",
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "default",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			options: provider.GetInstanceOptions{
				Name:      "test-app",
				Namespace: "default",
			},
			expectedCount: 0,
			wantErr:       false,
		},
		{
			name: "multiple pods with mixed states",
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-2",
						Namespace: "default",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
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

			// Create test pods
			for _, pod := range tt.pods {
				_, err := client.CoreV1().Pods(tt.options.Namespace).Create(context.Background(), &pod, metav1.CreateOptions{})
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

			// Verify instances are correct for ready pods
			for _, instance := range instances {
				assert.NotEmpty(t, instance.Address())
			}
		})
	}
}
func TestWatch(t *testing.T) {
	tests := []struct {
		name       string
		pods       []corev1.Pod
		options    provider.GetInstanceOptions
		operations []struct {
			op   watch.EventType
			pod  *corev1.Pod
			wait time.Duration
		}
		expectedEvents int
	}{
		{
			name: "new pod becomes ready",
			pods: []corev1.Pod{},
			options: provider.GetInstanceOptions{
				Name:      "test-app",
				Namespace: "default",
			},
			operations: []struct {
				op   watch.EventType
				pod  *corev1.Pod
				wait time.Duration
			}{
				{
					op: watch.Added,
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-pod-1",
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
					wait: 100 * time.Millisecond,
				},
			},
			expectedEvents: 1,
		},
		{
			name: "pod deleted",
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pod-1",
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
			operations: []struct {
				op   watch.EventType
				pod  *corev1.Pod
				wait time.Duration
			}{
				{
					op: watch.Deleted,
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-pod-1",
						},
					},
					wait: 100 * time.Millisecond,
				},
			},
			expectedEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()

			// Create initial pods
			for _, pod := range tt.pods {
				_, err := client.CoreV1().Pods(tt.options.Namespace).Create(context.Background(), &pod, metav1.CreateOptions{})
				assert.NoError(t, err)
			}

			k8sDiscovery := &K8sDiscovery{
				client: client,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			watchChan, err := k8sDiscovery.Watch(ctx, tt.options)
			assert.NoError(t, err)

			receivedEvents := 0
			done := make(chan bool)

			go func() {
				for range watchChan {
					receivedEvents++
				}
				done <- true
			}()

			// Perform operations
			for _, op := range tt.operations {
				switch op.op {
				case watch.Added:
					_, err := client.CoreV1().Pods(tt.options.Namespace).Create(ctx, op.pod, metav1.CreateOptions{})
					assert.NoError(t, err)
				case watch.Deleted:
					err := client.CoreV1().Pods(tt.options.Namespace).Delete(ctx, op.pod.Name, metav1.DeleteOptions{})
					assert.NoError(t, err)
				}
				time.Sleep(op.wait)
			}

			// Wait a bit for events to be processed
			time.Sleep(200 * time.Millisecond)
			cancel()
			<-done

			assert.Equal(t, tt.expectedEvents, receivedEvents)
		})
	}
}
