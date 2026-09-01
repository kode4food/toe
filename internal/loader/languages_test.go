package loader_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
)

func TestDefaultLanguages(t *testing.T) {
	langs, ok := loader.DefaultLanguages()

	assert.True(t, ok)
	assert.NotEmpty(t, langs["language"])
}
