package register_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/register"
)

func TestWriteRead(t *testing.T) {
	r := register.New()
	r.Write('"', []string{"a", "b", "c"})
	got := r.Read('"')
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestBlackHole(t *testing.T) {
	r := register.New()
	r.Write('_', []string{"x"})
	assert.Nil(t, r.Read('_'))
}

func TestFirst(t *testing.T) {
	r := register.New()
	r.Write('a', []string{"first", "second"})
	v, ok := r.First('a')
	assert.True(t, ok)
	assert.Equal(t, "first", v)
}

func TestFirstEmpty(t *testing.T) {
	r := register.New()
	_, ok := r.First('z')
	assert.False(t, ok)
}

func TestOverwrite(t *testing.T) {
	r := register.New()
	r.Write('"', []string{"old"})
	r.Write('"', []string{"new"})
	got := r.Read('"')
	assert.Equal(t, []string{"new"}, got)
}

func TestSet(t *testing.T) {
	r := register.New()
	r.Set('a', "hello")
	v, ok := r.First('a')
	assert.True(t, ok)
	assert.Equal(t, "hello", v)
}

func TestClear(t *testing.T) {
	r := register.New()
	r.Write('a', []string{"x"})
	r.Clear('a')
	assert.Nil(t, r.Read('a'))
}

func TestClearAll(t *testing.T) {
	r := register.New()
	r.Write('a', []string{"x"})
	r.Write('b', []string{"y"})
	r.ClearAll()
	assert.Nil(t, r.Read('a'))
	assert.Nil(t, r.Read('b'))
}
