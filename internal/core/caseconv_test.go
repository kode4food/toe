package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestToPascalCase(t *testing.T) {
	t.Run("converts snake_case to PascalCase", func(t *testing.T) {
		assert.Equal(t, "HelloWorld", core.ToPascalCase("hello_world"))
	})

	t.Run("converts space-separated words", func(t *testing.T) {
		assert.Equal(t, "FooBar", core.ToPascalCase("foo bar"))
	})

	t.Run("uppercases first char only of each word", func(t *testing.T) {
		assert.Equal(t, "HEllo", core.ToPascalCase("hEllo"))
	})

	t.Run("strips leading and trailing separators", func(t *testing.T) {
		assert.Equal(t, "Hello", core.ToPascalCase("_hello_"))
	})

	t.Run("already PascalCase", func(t *testing.T) {
		assert.Equal(t, "HelloWorld", core.ToPascalCase("HelloWorld"))
	})

	t.Run("handles empty string", func(t *testing.T) {
		assert.Equal(t, "", core.ToPascalCase(""))
	})

	t.Run("unicode word characters", func(t *testing.T) {
		assert.Equal(t, "Héllo", core.ToPascalCase("héllo"))
	})

	t.Run("multiple consecutive separators", func(t *testing.T) {
		assert.Equal(t, "FooBar", core.ToPascalCase("foo__bar"))
	})
}

func TestToCamelCase(t *testing.T) {
	t.Run("lowercases all alphanumeric chars", func(t *testing.T) {
		assert.Equal(t, "helloworld", core.ToCamelCase("hello_world"))
	})

	t.Run("strips non-alphanumeric chars", func(t *testing.T) {
		assert.Equal(t, "foobar", core.ToCamelCase("foo-bar"))
	})

	t.Run("lowercases uppercase input", func(t *testing.T) {
		assert.Equal(t, "helloworld", core.ToCamelCase("HelloWorld"))
	})

	t.Run("handles empty string", func(t *testing.T) {
		assert.Equal(t, "", core.ToCamelCase(""))
	})

	t.Run("unicode uppercase", func(t *testing.T) {
		assert.Equal(t, "héllo", core.ToCamelCase("Héllo"))
	})
}

func TestToUpperCase(t *testing.T) {
	t.Run("uppercases ASCII", func(t *testing.T) {
		assert.Equal(t, "HELLO", core.ToUpperCase("hello"))
	})

	t.Run("uppercases unicode", func(t *testing.T) {
		assert.Equal(t, "HÉLLO", core.ToUpperCase("héllo"))
	})

	t.Run("leaves digits unchanged", func(t *testing.T) {
		assert.Equal(t, "ABC123", core.ToUpperCase("abc123"))
	})

	t.Run("handles empty string", func(t *testing.T) {
		assert.Equal(t, "", core.ToUpperCase(""))
	})
}

func TestToLowerCase(t *testing.T) {
	t.Run("lowercases ASCII", func(t *testing.T) {
		assert.Equal(t, "hello", core.ToLowerCase("HELLO"))
	})

	t.Run("lowercases unicode", func(t *testing.T) {
		assert.Equal(t, "héllo", core.ToLowerCase("HÉLLO"))
	})

	t.Run("leaves digits unchanged", func(t *testing.T) {
		assert.Equal(t, "abc123", core.ToLowerCase("ABC123"))
	})

	t.Run("handles empty string", func(t *testing.T) {
		assert.Equal(t, "", core.ToLowerCase(""))
	})
}
