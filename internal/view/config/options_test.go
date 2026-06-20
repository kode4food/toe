package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/config"
)

func TestParseHelpers(t *testing.T) {
	t.Run("ParseBool true", func(t *testing.T) {
		v, err := config.ParseBool("true")
		assert.NoError(t, err)
		assert.True(t, v)
	})

	t.Run("ParseBool false", func(t *testing.T) {
		v, err := config.ParseBool("false")
		assert.NoError(t, err)
		assert.False(t, v)
	})

	t.Run("ParseBool invalid", func(t *testing.T) {
		_, err := config.ParseBool("nope")
		assert.ErrorIs(t, err, config.ErrInvalidOption)
	})

	t.Run("ParseNonNegInt zero", func(t *testing.T) {
		v, err := config.ParseNonNegInt("0")
		assert.NoError(t, err)
		assert.Equal(t, 0, v)
	})

	t.Run("ParseNonNegInt negative rejected", func(t *testing.T) {
		_, err := config.ParseNonNegInt("-1")
		assert.ErrorIs(t, err, config.ErrInvalidOption)
	})

	t.Run("ParsePositiveInt one", func(t *testing.T) {
		v, err := config.ParsePositiveInt("1")
		assert.NoError(t, err)
		assert.Equal(t, 1, v)
	})

	t.Run("ParsePositiveInt zero rejected", func(t *testing.T) {
		_, err := config.ParsePositiveInt("0")
		assert.ErrorIs(t, err, config.ErrInvalidOption)
	})

	t.Run("ParseIntSlice", func(t *testing.T) {
		v, err := config.ParseIntSlice("[80, 120]")
		assert.NoError(t, err)
		assert.Equal(t, []int{80, 120}, v)
	})

	t.Run("ParseIntSlice invalid", func(t *testing.T) {
		_, err := config.ParseIntSlice("not a slice")
		assert.ErrorIs(t, err, config.ErrInvalidOption)
	})

	t.Run("ParseStringSlice", func(t *testing.T) {
		v, err := config.ParseStringSlice(`["bash", "-c"]`)
		assert.NoError(t, err)
		assert.Equal(t, []string{"bash", "-c"}, v)
	})

	t.Run("ParseStringSlice invalid", func(t *testing.T) {
		_, err := config.ParseStringSlice("not a slice")
		assert.ErrorIs(t, err, config.ErrInvalidOption)
	})

	t.Run("ParseStringLiteral plain value", func(t *testing.T) {
		v, err := config.ParseStringLiteral("hello")
		assert.NoError(t, err)
		assert.Equal(t, "hello", v)
	})

	t.Run("ParseStringLiteral quoted string", func(t *testing.T) {
		v, err := config.ParseStringLiteral(`"hello world"`)
		assert.NoError(t, err)
		assert.Equal(t, "hello world", v)
	})

	t.Run("ParseStringLiteral single-quoted string", func(t *testing.T) {
		v, err := config.ParseStringLiteral(`'hello'`)
		assert.NoError(t, err)
		assert.Equal(t, "hello", v)
	})

	t.Run("ParseStringLiteral empty returns empty", func(t *testing.T) {
		v, err := config.ParseStringLiteral("  ")
		assert.NoError(t, err)
		assert.Equal(t, "", v)
	})

	t.Run("ParseStringLiteral invalid quoted errors", func(t *testing.T) {
		_, err := config.ParseStringLiteral(`"unclosed`)
		assert.ErrorIs(t, err, config.ErrInvalidOption)
	})

	t.Run("FormatIntSlice", func(t *testing.T) {
		s := config.FormatIntSlice([]int{80, 120})
		assert.Equal(t, "[80, 120]", s)
	})

	t.Run("FormatStringSlice", func(t *testing.T) {
		s := config.FormatStringSlice([]string{"bash", "-c"})
		assert.Equal(t, `["bash", "-c"]`, s)
	})

}
