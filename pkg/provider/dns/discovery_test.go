package dns

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/stretchr/testify/assert"
)

func TestDNSServiceDiscovery_GetInstances(t *testing.T) {
	tests := []struct {
		name        string
		servers     []string
		serviceName string
		valid       time.Duration
		mockIPs     []string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid service with port",
			servers:     []string{"8.8.8.8:53"},
			serviceName: "www.google.com:8080",
			valid:       time.Duration(0),
			mockIPs:     []string{"142.250.198.68:8080"},
			wantErr:     false,
		},
		{
			name:        "valid service without port",
			servers:     []string{"1.1.1.1"},
			serviceName: "www.google.com",
			valid:       time.Duration(10 * time.Minute),
			mockIPs:     []string{"142.250.198.68"},
			wantErr:     false,
		},
		{
			name:        "localhost service",
			servers:     []string{"8.8.8.8:53"},
			serviceName: "localhost:8080",
			valid:       time.Duration(0),
			mockIPs:     []string{"127.0.0.1"},
			wantErr:     false,
		},
		{
			name:        "service not found",
			servers:     []string{"8.8.8.8:53"},
			serviceName: "nonexistent.com",
			valid:       time.Duration(0),
			mockIPs:     []string{},
			wantErr:     true,
			errMsg:      "dns: no records found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDNSServiceDiscovery(tt.servers, tt.valid)
			assert.NoError(t, err)

			ctx := context.Background()

			options := provider.GetInstanceOptions{
				ID: tt.serviceName,
			}

			got, err := d.GetInstances(ctx, options)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetInstances() expected error")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("GetInstances() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("GetInstances() unexpected error: %v", err)
				return
			}

			if len(got) < len(tt.mockIPs) {
				t.Errorf("GetInstances() got %d instances, want %d", len(got), len(tt.mockIPs))
			}
		})
	}
}
