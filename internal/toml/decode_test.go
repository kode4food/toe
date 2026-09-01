package toml_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/toml"
)

func TestValuePointers(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		v := toml.BoolPtr(true)

		assert.NotNil(t, v)
		assert.Equal(t, true, *v)
		assert.Nil(t, toml.BoolPtr("true"))
	})

	t.Run("int", func(t *testing.T) {
		v := toml.IntPtr(3)

		assert.NotNil(t, v)
		assert.Equal(t, 3, *v)

		v = toml.IntPtr(int64(4))

		assert.NotNil(t, v)
		assert.Equal(t, 4, *v)

		assert.Nil(t, toml.IntPtr("4"))
	})

	t.Run("string", func(t *testing.T) {
		v := toml.StringPtr("go")

		assert.NotNil(t, v)
		assert.Equal(t, "go", *v)
		assert.Nil(t, toml.StringPtr(1))
	})
}

func TestAnySlice(t *testing.T) {
	t.Run("[]any passthrough", func(t *testing.T) {
		in := []any{1, 2, 3}
		out, ok := toml.AnySlice(in)
		assert.True(t, ok)
		assert.Equal(t, in, out)
	})

	t.Run("[]map[string]any converts", func(t *testing.T) {
		in := []map[string]any{{"a": 1}, {"b": 2}}
		out, ok := toml.AnySlice(in)
		assert.True(t, ok)
		assert.Equal(t, []any{
			map[string]any{"a": 1}, map[string]any{"b": 2},
		}, out)
	})

	t.Run("[]string converts", func(t *testing.T) {
		in := []string{"x", "y"}
		out, ok := toml.AnySlice(in)
		assert.True(t, ok)
		assert.Equal(t, []any{"x", "y"}, out)
	})

	t.Run("unknown type returns false", func(t *testing.T) {
		_, ok := toml.AnySlice(42)
		assert.False(t, ok)
	})
}

func TestMapValues(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		m := map[string]any{"name": "go", "count": 2}

		assert.Equal(t, "go", toml.StringValue(m, "name"))
		assert.Equal(t, "", toml.StringValue(m, "count"))
		assert.Equal(t, "", toml.StringValue(m, "missing"))
	})

	t.Run("int", func(t *testing.T) {
		m := map[string]any{"width": int64(8), "name": "go"}

		assert.Equal(t, 8, toml.IntValue(m, "width", 4))
		assert.Equal(t, 4, toml.IntValue(m, "name", 4))
		assert.Equal(t, 4, toml.IntValue(m, "missing", 4))
	})
}

func TestStringSlices(t *testing.T) {
	t.Run("bare string", func(t *testing.T) {
		assert.Equal(t, []string{"go"}, toml.StringOrSlice("go"))
	})

	t.Run("slice", func(t *testing.T) {
		assert.Equal(t, []string{"go", "mod"}, toml.StringOrSlice(
			[]any{"go", "mod"},
		))
	})

	t.Run("drops non-strings", func(t *testing.T) {
		assert.Equal(t, []string{"go"}, toml.StringSlice([]any{"go", 1}))
	})

	t.Run("not a slice", func(t *testing.T) {
		assert.Nil(t, toml.StringSlice(42))
		assert.Nil(t, toml.StringOrSlice(42))
	})
}

func TestMaps(t *testing.T) {
	t.Run("string map drops non-strings", func(t *testing.T) {
		out := toml.StringMap(map[string]any{"a": "1", "b": 2})

		assert.Equal(t, map[string]string{"a": "1"}, out)
	})

	t.Run("any map passthrough", func(t *testing.T) {
		in := map[string]any{"a": 1}

		assert.Equal(t, in, toml.AnyMap(in))
	})

	t.Run("not a table", func(t *testing.T) {
		assert.Nil(t, toml.StringMap("nope"))
		assert.Nil(t, toml.AnyMap("nope"))
	})
}
