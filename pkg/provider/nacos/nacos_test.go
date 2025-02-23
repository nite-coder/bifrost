package nacos

import (
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockConfigClient struct {
	mock.Mock
}

func (m *MockConfigClient) GetConfig(param vo.ConfigParam) (string, error) {
	args := m.Called(param)
	return args.String(0), args.Error(1)
}

func (m *MockConfigClient) PublishConfig(param vo.ConfigParam) (bool, error) {
	args := m.Called(param)
	return args.Bool(0), args.Error(1)
}

func (m *MockConfigClient) DeleteConfig(param vo.ConfigParam) (bool, error) {
	args := m.Called(param)
	return args.Bool(0), args.Error(1)
}

func (m *MockConfigClient) ListenConfig(param vo.ConfigParam) error {
	args := m.Called(param)
	return args.Error(0)
}

func (m *MockConfigClient) CancelListenConfig(param vo.ConfigParam) error {
	args := m.Called(param)
	return args.Error(0)
}

func (m *MockConfigClient) SearchConfig(param vo.SearchConfigParam) (*model.ConfigPage, error) {
	args := m.Called(param)
	return args.Get(0).(*model.ConfigPage), args.Error(1)
}

func (m *MockConfigClient) CloseClient() {
	m.Called()
}

func TestNewProvider(t *testing.T) {
	opts := Options{
		Config: Config{
			NamespaceID: "test-namespace",
			ContextPath: "/custom-path",
			LogDir:      "/custom-log",
			CacheDir:    "/custom-cache",
			Timeout:     3 * time.Second,
			Endpoints:   []string{"http://localhost:8848"},
			Files: []*File{
				{DataID: "test-data-id", Group: "test-group"},
			},
		},
	}

	provider, err := NewProvider(opts)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, opts, provider.options)
}

func TestConfigOpen(t *testing.T) {
	mockClient := new(MockConfigClient)
	mockClient.On("GetConfig", vo.ConfigParam{
		DataId: "test-data-id",
		Group:  "test-group",
	}).Return("test-content", nil)

	opts := Options{
		Config: Config{
			Files: []*File{
				{DataID: "test-data-id", Group: "test-group"},
			},
		},
	}

	provider := &NacosProvider{
		client:  mockClient,
		options: opts,
	}

	files, err := provider.ConfigOpen()
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "test-data-id", files[0].DataID)
	assert.Equal(t, "test-group", files[0].Group)
	assert.Equal(t, "test-content", files[0].Content)

	mockClient.AssertExpectations(t)
}

func TestSetOnChanged(t *testing.T) {
	var called bool
	changeFunc := func() error {
		called = true
		return nil
	}

	opts := Options{
		Config: Config{},
	}

	provider := &NacosProvider{
		options: opts,
	}

	provider.SetOnChanged(changeFunc)
	assert.NotNil(t, provider.OnChanged)

	err := provider.OnChanged()
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestWatch(t *testing.T) {
	mockClient := new(MockConfigClient)

	mockClient.On("ListenConfig", mock.MatchedBy(func(param vo.ConfigParam) bool {
		return param.DataId == "test-data-id" &&
			param.Group == "test-group" &&
			param.OnChange != nil
	})).Return(nil)

	opts := Options{
		Config: Config{
			Watch: true,
			Files: []*File{
				{DataID: "test-data-id", Group: "test-group"},
			},
		},
	}

	provider := &NacosProvider{
		client:  mockClient,
		options: opts,
	}

	err := provider.Watch()
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}
