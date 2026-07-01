package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/ui"
)

func TestCompletionIconModeUnmarshal(t *testing.T) {
	t.Run("empty defaults to codicon", func(t *testing.T) {
		var m ui.CompletionIconMode
		assert.NoError(t, m.UnmarshalText([]byte("")))
		assert.Equal(t, ui.CompletionIconsCodicon, m)
	})

	t.Run("valid value accepted", func(t *testing.T) {
		var m ui.CompletionIconMode
		assert.NoError(t, m.UnmarshalText([]byte("ascii")))
		assert.Equal(t, ui.CompletionIconsASCII, m)
	})

	t.Run("invalid value errors", func(t *testing.T) {
		var m ui.CompletionIconMode
		assert.Error(t, m.UnmarshalText([]byte("bogus")))
	})
}
