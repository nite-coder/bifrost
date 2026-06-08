package optional

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOption_CreationAndUnwrap(t *testing.T) {
	// Test Some
	opt1 := Some("hello")
	assert.True(t, opt1.IsSome())
	assert.False(t, opt1.IsNone())
	assert.Equal(t, "hello", opt1.Unwrap())
	assert.Equal(t, "hello", opt1.UnwrapOr("world"))

	// Test None
	opt2 := None[int]()
	assert.False(t, opt2.IsSome())
	assert.True(t, opt2.IsNone())
	assert.Equal(t, 99, opt2.UnwrapOr(99))
	assert.Panics(t, func() {
		opt2.Unwrap()
	})
}

func TestOption_JSONMarshal(t *testing.T) {
	type TestStruct struct {
		Name  string         `json:"name"`
		Param Option[int]    `json:"param"`
		Code  Option[string] `json:"code"`
	}

	t.Run("with values", func(t *testing.T) {
		ts := TestStruct{
			Name:  "test1",
			Param: Some(42),
			Code:  Some("err_code"),
		}

		data, err := json.Marshal(ts)
		require.NoError(t, err)
		assert.JSONEq(t, `{"name":"test1","param":42,"code":"err_code"}`, string(data))
	})

	t.Run("with none values", func(t *testing.T) {
		ts := TestStruct{
			Name:  "test2",
			Param: None[int](),
			Code:  None[string](),
		}

		data, err := json.Marshal(ts)
		require.NoError(t, err)
		assert.JSONEq(t, `{"name":"test2","param":null,"code":null}`, string(data))
	})
}

func TestOption_JSONUnmarshal(t *testing.T) {
	type TestStruct struct {
		Name  string         `json:"name"`
		Param Option[int]    `json:"param"`
		Code  Option[string] `json:"code"`
	}

	t.Run("with values", func(t *testing.T) {
		jsonStr := `{"name":"test1","param":42,"code":"err_code"}`
		var ts TestStruct

		err := json.Unmarshal([]byte(jsonStr), &ts)
		require.NoError(t, err)

		assert.Equal(t, "test1", ts.Name)
		assert.True(t, ts.Param.IsSome())
		assert.Equal(t, 42, ts.Param.Unwrap())
		assert.True(t, ts.Code.IsSome())
		assert.Equal(t, "err_code", ts.Code.Unwrap())
	})

	t.Run("with null values", func(t *testing.T) {
		jsonStr := `{"name":"test2","param":null,"code":null}`
		var ts TestStruct

		err := json.Unmarshal([]byte(jsonStr), &ts)
		require.NoError(t, err)

		assert.Equal(t, "test2", ts.Name)
		assert.True(t, ts.Param.IsNone())
		assert.True(t, ts.Code.IsNone())
	})

	t.Run("with missing fields", func(t *testing.T) {
		// If fields are missing in JSON, they remain in their zero state (which is None)
		jsonStr := `{"name":"test3"}`
		var ts TestStruct

		err := json.Unmarshal([]byte(jsonStr), &ts)
		require.NoError(t, err)

		assert.Equal(t, "test3", ts.Name)
		assert.True(t, ts.Param.IsNone())
		assert.True(t, ts.Code.IsNone())
	})
}
