package grpc

import (
	"fmt"
)

type rawCodec struct{}

func (c *rawCodec) Marshal(v any) ([]byte, error) {
	return v.([]byte), nil
}

func (c *rawCodec) Unmarshal(data []byte, v any) error {
	switch dst := v.(type) {
	case *[]byte:
		*dst = data
	case []byte:
		copy(dst, data)
	default:
		return fmt.Errorf("rawCodec.Unmarshal: unsupported type: %T", v)
	}

	return nil
}

func (c *rawCodec) Name() string {
	return "raw"
}
