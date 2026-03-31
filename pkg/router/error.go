package router

import "errors"

// ErrAlreadyExists is returned when a route already exists.
var ErrAlreadyExists = errors.New("already exists")
