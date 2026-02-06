package nacos

import (
	"context"
	"fmt"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockNamingClient struct {
	mock.Mock
}

func (m *MockNamingClient) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	args := m.Called(param)
	return args.Bool(0), args.Error(1)
}

func (m *MockNamingClient) BatchRegisterInstance(param vo.BatchRegisterInstanceParam) (bool, error) {
	args := m.Called(param)
	return args.Bool(0), args.Error(1)
}

func (m *MockNamingClient) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	args := m.Called(param)
	return args.Bool(0), args.Error(1)
}

func (m *MockNamingClient) UpdateInstance(param vo.UpdateInstanceParam) (bool, error) {
	args := m.Called(param)
	return args.Bool(0), args.Error(1)
}

func (m *MockNamingClient) GetService(param vo.GetServiceParam) (model.Service, error) {
	args := m.Called(param)
	return args.Get(0).(model.Service), args.Error(1)
}

func (m *MockNamingClient) SelectAllInstances(param vo.SelectAllInstancesParam) ([]model.Instance, error) {
	args := m.Called(param)
	return args.Get(0).([]model.Instance), args.Error(1)
}

func (m *MockNamingClient) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	args := m.Called(param)
	return args.Get(0).([]model.Instance), args.Error(1)
}

func (m *MockNamingClient) SelectOneHealthyInstance(param vo.SelectOneHealthInstanceParam) (*model.Instance, error) {
	args := m.Called(param)
	return args.Get(0).(*model.Instance), args.Error(1)
}

func (m *MockNamingClient) Subscribe(param *vo.SubscribeParam) error {
	args := m.Called(param)
	return args.Error(0)
}

func (m *MockNamingClient) Unsubscribe(param *vo.SubscribeParam) error {
	args := m.Called(param)
	return args.Error(0)
}

func (m *MockNamingClient) GetAllServicesInfo(param vo.GetAllServiceInfoParam) (model.ServiceList, error) {
	args := m.Called(param)
	return args.Get(0).(model.ServiceList), args.Error(1)
}

func (m *MockNamingClient) ServerHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockNamingClient) CloseClient() {
	m.Called()
}

func TestNacosServiceDiscovery_GetInstances(t *testing.T) {
	tests := []struct {
		name          string
		mockSetup     func(*MockNamingClient)
		options       provider.GetInstanceOptions
		want          int
		wantErr       bool
		metadataCount int
	}{
		{
			name: "successful get instances",
			mockSetup: func(m *MockNamingClient) {
				m.On("SelectInstances", mock.MatchedBy(func(param vo.SelectInstancesParam) bool {
					return param.ServiceName == "test-service"
				})).Return([]model.Instance{
					{
						Ip:     "127.0.0.1",
						Port:   8080,
						Weight: 1,
						Metadata: map[string]string{
							"id":   "123",
							"name": "angela",
						},
					},
					{
						Ip:     "127.0.0.2",
						Port:   8081,
						Weight: 2,
					},
				}, nil)
			},
			options: provider.GetInstanceOptions{
				Name:  "test-service",
				Group: "test-group",
			},
			want:          2,
			wantErr:       false,
			metadataCount: 2,
		},
		{
			name: "empty instance list",
			mockSetup: func(m *MockNamingClient) {
				m.On("SelectInstances", mock.MatchedBy(func(param vo.SelectInstancesParam) bool {
					return param.ServiceName == "test-service"
				})).Return([]model.Instance{}, nil)
			},
			options: provider.GetInstanceOptions{
				Name:  "test-service",
				Group: "test-group",
			},
			want:          0,
			wantErr:       false,
			metadataCount: 0,
		},
		{
			name: "nacos error",
			mockSetup: func(m *MockNamingClient) {
				m.On("SelectInstances", mock.MatchedBy(func(param vo.SelectInstancesParam) bool {
					return param.ServiceName == "test-service"
				})).Return([]model.Instance{}, fmt.Errorf("nacos error"))
			},
			options: provider.GetInstanceOptions{
				Name:  "test-service",
				Group: "test-group",
			},
			want:          0,
			wantErr:       true,
			metadataCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockNamingClient{}
			tt.mockSetup(mockClient)

			// mock newNamingClientFunc
			originalFunc := newNamingClientFunc
			defer func() { newNamingClientFunc = originalFunc }()
			newNamingClientFunc = func(param vo.NacosClientParam) (naming_client.INamingClient, error) {
				// We expect the connectivity check
				mockClient.On("GetAllServicesInfo", mock.MatchedBy(func(param vo.GetAllServiceInfoParam) bool {
					return param.PageNo == 1 && param.PageSize == 1
				})).Return(model.ServiceList{}, nil)
				return mockClient, nil
			}

			d, err := NewNacosServiceDiscovery(Options{
				Endpoints: []string{"http://127.0.0.1:8848"},
			})
			assert.NoError(t, err)

			// d.client is already set correctly by our mock constructor
			// d.client = mockClient

			got, err := d.GetInstances(context.Background(), tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.metadataCount > 0 {
				id, found := got[0].Tag("id")
				assert.True(t, found)
				assert.Equal(t, "123", id)
			}

			assert.Equal(t, tt.want, len(got))
		})
	}
}
func TestNacosServiceDiscovery_Watch(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(*MockNamingClient)
		options   provider.GetInstanceOptions
		wantErr   bool
	}{
		{
			name: "successful watch",
			mockSetup: func(m *MockNamingClient) {
				m.On("Subscribe", mock.MatchedBy(func(param *vo.SubscribeParam) bool {
					return param.ServiceName == "test-service" && param.GroupName == "test-group"
				})).Return(nil)

			},
			options: provider.GetInstanceOptions{
				Name:  "test-service",
				Group: "test-group",
			},
			wantErr: false,
		},
		{
			name: "subscribe error",
			mockSetup: func(m *MockNamingClient) {
				m.On("Subscribe", mock.Anything).Return(fmt.Errorf("subscription error"))
			},
			options: provider.GetInstanceOptions{
				Name:  "test-service",
				Group: "test-group",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockNamingClient{}
			tt.mockSetup(mockClient)

			// mock newNamingClientFunc
			originalFunc := newNamingClientFunc
			defer func() { newNamingClientFunc = originalFunc }()
			newNamingClientFunc = func(param vo.NacosClientParam) (naming_client.INamingClient, error) {
				// Check connection expectation
				mockClient.On("GetAllServicesInfo", mock.MatchedBy(func(param vo.GetAllServiceInfoParam) bool {
					return param.PageNo == 1 && param.PageSize == 1
				})).Return(model.ServiceList{}, nil)
				return mockClient, nil
			}

			d, err := NewNacosServiceDiscovery(Options{
				Endpoints: []string{"http://127.0.0.1:8848"},
			})
			assert.NoError(t, err)

			// d.client = mockClient

			ch, err := d.Watch(context.Background(), tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("Watch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.NotNil(t, ch)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
