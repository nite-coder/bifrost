package gateway

import (
	"errors"
)

var (
	// ErrConfigNotFound is returned when the configuration is not found.
	ErrConfigNotFound = errors.New("config not found")
	// ErrAlreadyExists is returned when a resource already exists.
	ErrAlreadyExists = errors.New("already exists")
	// ErrNoLiveUpstream is returned when no live upstreams are available.
	ErrNoLiveUpstream = errors.New("no live upstreams while connecting to upstream")
)
