package grpc

import (
	"fmt"

	"google.golang.org/grpc/encoding"
)

type rawCodec struct{}

func (c *rawCodec) Marshal(v interface{}) ([]byte, error) {
	return v.([]byte), nil
}

func (c *rawCodec) Unmarshal(data []byte, v interface{}) error {
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

func init() {
	encoding.RegisterCodec(&rawCodec{})
}
