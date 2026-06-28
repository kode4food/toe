package lsp_test

import (
	"errors"
	"testing"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view/language"
	"github.com/stretchr/testify/assert"
)

func TestRegistry(t *testing.T) {
	t.Run("copies servers", func(t *testing.T) {
		servers := map[string]language.Server{
			"go": {Command: "gopls"},
		}
		reg := lsp.NewRegistry(servers)
		servers["go"] = language.Server{Command: "changed"}

		got, ok := reg.Server("go")

		assert.True(t, ok)
		assert.Equal(t, "gopls", got.Command)
	})

	t.Run("rejects duplicate", func(t *testing.T) {
		reg := lsp.NewRegistry(map[string]language.Server{
			"go": {Command: "gopls"},
		})

		err := reg.Register("go", language.Server{Command: "other"})

		assert.True(t, errors.Is(err, lsp.ErrServerExists))
	})

	t.Run("registers server", func(t *testing.T) {
		reg := lsp.NewRegistry(nil)

		err := reg.Register("go", language.Server{Command: "gopls"})
		got, ok := reg.Server("go")

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "gopls", got.Command)
	})
}
