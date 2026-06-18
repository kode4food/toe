package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestJumplistPicker(t *testing.T) {
	t.Run("lists jumps with line contents", func(t *testing.T) {
		m, _, _ := jumplistModel(t)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "file.txt")
		assert.Contains(t, out, "TARGET")
	})

	t.Run("accept moves cursor to the jump", func(t *testing.T) {
		m, e, anchor := jumplistModel(t)
		m = sendSpecial(m, tea.KeyEnter)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
		assert.Equal(t, anchor, pos)
	})
}
