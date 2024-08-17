package gateway

import (
	"errors"
)

var (
	ErrConfigNotFound              = errors.New("config not found")
	ErrAlreadyExists               = errors.New("already exists")
	ErrNoLiveUpstream              = errors.New("no live upstreams while connecting to upstream")
	ErrUpstreamTemporarilyDisabled = errors.New("upstream server temporarily disabled")
)
