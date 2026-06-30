package lsp_test

import (
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
}
