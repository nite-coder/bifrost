package gateway

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/target"
)

var _ proxy.Proxy = (*mockProxyForUpdate)(nil)

type mockProxyForUpdate struct {
	id         string
	target     string
	ep         *target.Endpoint
	onClose    func()
	setEpCount int
}

func (m *mockProxyForUpdate) ID() string                                         { return m.id }
func (m *mockProxyForUpdate) Target() string                                     { return m.target }
func (m *mockProxyForUpdate) Endpoint() *target.Endpoint                         { return m.ep }
func (m *mockProxyForUpdate) SetEndpoint(ep *target.Endpoint)                    { m.ep = ep; m.setEpCount++ }
func (m *mockProxyForUpdate) ServeHTTP(_ context.Context, _ *app.RequestContext) {}
func (m *mockProxyForUpdate) Close() error {
	if m.onClose != nil {
		m.onClose()
	}
	return nil
}
