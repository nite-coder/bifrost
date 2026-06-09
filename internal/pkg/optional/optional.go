package optional

import (
	"bytes"
	"encoding/json"
)

var jsonNull = []byte("null")

// Option represents an optional value of type T.
// It can either be empty (None) or contain a value (Some).
// When marshaled to JSON, an empty Option becomes `null`.
type Option[T any] struct {
	value    T
	hasValue bool
}

// Some creates an Option containing the given value.
func Some[T any](v T) Option[T] {
	return Option[T]{
		value:    v,
		hasValue: true,
	}
}

// None creates an empty Option of type T.
func None[T any]() Option[T] {
	return Option[T]{
		hasValue: false,
	}
}

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool {
	return o.hasValue
}

// IsNone returns true if the Option is empty.
func (o Option[T]) IsNone() bool {
	return !o.hasValue
}

// Unwrap returns the inner value. It panics if the Option is empty.
func (o Option[T]) Unwrap() T {
	if !o.hasValue {
		panic("called Unwrap on an empty Option")
	}
	return o.value
}

// UnwrapOr returns the inner value if present, otherwise returns the provided default value.
func (o Option[T]) UnwrapOr(defaultValue T) T {
	if !o.hasValue {
		return defaultValue
	}
	return o.value
}

// MarshalJSON implements the json.Marshaler interface.
// An empty Option marshals to literal "null".
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.hasValue {
		return jsonNull, nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It correctly handles JSON "null" strings.
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), jsonNull) {
		o.hasValue = false
		var zero T
		o.value = zero
		return nil
	}

	err := json.Unmarshal(data, &o.value)
	if err != nil {
		return err
	}

	o.hasValue = true
	return nil
}
