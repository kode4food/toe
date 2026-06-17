package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

func TestFromTeaKey(t *testing.T) {
	t.Run("page keys ignore text fallback", func(t *testing.T) {
		up := ui.FromTeaKey(tea.KeyPressMsg{
			Code: tea.KeyPgUp,
			Text: "pgup",
		})
		down := ui.FromTeaKey(tea.KeyPressMsg{
			Code: tea.KeyPgDown,
			Text: "pgdown",
		})

		assert.Equal(t, command.Special("pageup"), up)
		assert.Equal(t, command.Special("pagedown"), down)
	})
}
