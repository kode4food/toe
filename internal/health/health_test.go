package health_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/health"
)

func TestHealth(t *testing.T) {
	t.Run("checks bundled runtime", func(t *testing.T) {
		rep := health.CheckRuntime()

		assert.True(t, rep.OK())
		assert.Len(t, rep, 4)
	})

	t.Run("writes report", func(t *testing.T) {
		var b bytes.Buffer

		err := health.Run(&b)

		assert.NoError(t, err)
		assert.Contains(t, b.String(), "toe health: ok")
		assert.Contains(t, b.String(), "- languages: ok")
		assert.Contains(t, b.String(), "- themes: ok")
	})
}
