package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestCommandPalettePicker(t *testing.T) {
	t.Run("lists registered commands", func(t *testing.T) {
		m, _ := paletteModel(t)
		assert.Contains(t, stripANSI(m.View().Content), "palette_probe")
	})

	t.Run("accepts and runs the command", func(t *testing.T) {
		m, e := paletteModel(t)
		for _, ch := range "palette_probe" {
			m = sendKey(m, ch)
		}
		m = sendSpecial(m, tea.KeyEnter)
		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}
