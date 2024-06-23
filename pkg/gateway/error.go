package gateway

import (
	"errors"
)

var (
	ErrConfigNotFound = errors.New("config not found")
	ErrAlreadyExists  = errors.New("already exists")
)
