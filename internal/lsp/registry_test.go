package lsp_test

import (
	"errors"
	"os"
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

	t.Run("looks up client by id", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		reg := lsp.NewRegistry(map[string]language.Server{
			"test": {
				Command: exe,
				Args:    []string{"-test.run=TestLSPServerProcess"},
				Environment: map[string]string{
					testServerEnv: "1",
				},
			},
		})

		id, client, err := reg.Start(t.Context(), "test", "", nil)
		assert.NoError(t, err)
		defer client.Close()

		got, ok := reg.Client(id)

		assert.True(t, ok)
		assert.Same(t, client, got)
	})

	t.Run("returns false for missing id", func(t *testing.T) {
		reg := lsp.NewRegistry(nil)

		_, ok := reg.Client(0)

		assert.False(t, ok)
	})

	t.Run("looks up clients by name", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		reg := lsp.NewRegistry(map[string]language.Server{
			"test": {
				Command: exe,
				Args:    []string{"-test.run=TestLSPServerProcess"},
				Environment: map[string]string{
					testServerEnv: "1",
				},
			},
		})

		_, client, err := reg.Start(t.Context(), "test", "", nil)
		assert.NoError(t, err)
		defer client.Close()

		clients := reg.Clients("test")

		assert.Len(t, clients, 1)
		assert.Same(t, client, clients[0])
	})

	t.Run("empty for missing name", func(t *testing.T) {
		reg := lsp.NewRegistry(nil)

		clients := reg.Clients("missing")

		assert.Empty(t, clients)
	})
}
