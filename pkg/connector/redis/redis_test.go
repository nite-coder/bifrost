package redis

import (
	"context"
	"testing"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name    string
		options []config.RedisOptions
		wantErr bool
	}{
		{
			name:    "empty options",
			options: []config.RedisOptions{},
			wantErr: false,
		},
		{
			name: "options with empty addrs",
			options: []config.RedisOptions{
				{
					Addrs: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "options with empty id",
			options: []config.RedisOptions{
				{
					Addrs: []string{"localhost:6379"},
					ID:    "",
				},
			},
			wantErr: true,
		},
		{
			name: "single redis with empty addr",
			options: []config.RedisOptions{
				{
					Addrs: []string{""},
					ID:    "test",
				},
			},
			wantErr: true,
		},
		{
			name: "single redis with invalid ping",
			options: []config.RedisOptions{
				{
					Addrs: []string{"localhost:6380"},
					ID:    "test",
				},
			},
			wantErr: true,
		},
		{
			name: "multi redis with empty addr",
			options: []config.RedisOptions{
				{
					Addrs: []string{"", "localhost:6379"},
					ID:    "test",
				},
			},
			wantErr: true,
		},
		{
			name: "multi redis with invalid ping",
			options: []config.RedisOptions{
				{
					Addrs: []string{"localhost:6379", "localhost:6380"},
					ID:    "test",
				},
			},
			wantErr: true,
		},
		{
			name: "valid single redis",
			options: []config.RedisOptions{
				{
					Addrs:    []string{"localhost:6379"},
					ID:       "test",
					SkipPing: true,
				},
			},
			wantErr: false,
		},
		{
			name: "valid multi redis",
			options: []config.RedisOptions{
				{
					Addrs:    []string{"localhost:6379", "localhost:6380"},
					ID:       "test",
					SkipPing: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(context.Background(), tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitializeSingleRedis(t *testing.T) {
	options := []config.RedisOptions{
		{
			Addrs:    []string{"localhost:6379"},
			ID:       "test",
			SkipPing: true,
		},
	}

	err := Initialize(context.Background(), options)
	assert.NoError(t, err)

	client, ok := Get("test")
	assert.True(t, ok)
	assert.NotNil(t, client)

	client, found := Get("test2")
	assert.False(t, found)
	assert.Nil(t, client)
}

func TestInitializeMultiRedis(t *testing.T) {
	options := []config.RedisOptions{
		{
			Addrs:    []string{"localhost:6379", "localhost:6380"},
			ID:       "test",
			SkipPing: true,
		},
	}

	err := Initialize(context.Background(), options)
	assert.NoError(t, err)

	client, ok := Get("test")
	assert.True(t, ok)
	assert.NotNil(t, client)
}
