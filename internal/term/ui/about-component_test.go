package ui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestAbout(t *testing.T) {
	open := func(t *testing.T, size geom.Size) ui.Model {
		t.Helper()
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, size.Width, size.Height)
		m = sendKey(m, ':')
		m = typeString(m, "about")
		return sendSpecial(m, tea.KeyEnter)
	}

	t.Run("shows name and license", func(t *testing.T) {
		m := open(t, geom.Size{Width: 80, Height: 24})
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "Thom's Own Editor")
		assert.Contains(t, out, "MIT License")
		assert.Contains(t, out, "https://github.com/kode4food/toe")
		assert.Contains(t, out, "A modal text editor for Go development")
		assert.Contains(t, out, i18n.Text(i18n.AboutDevelopment))
		assert.NotContains(t,
			out,
			i18n.Text(i18n.AboutVersion)+" "+i18n.Text(i18n.AboutDevelopment),
		)
		assert.Contains(t, out, "go1.")
	})

	t.Run("dismissed by a key press", func(t *testing.T) {
		m := open(t, geom.Size{Width: 80, Height: 24})
		assert.Contains(t, stripANSI(m.View().Content), "MIT License")

		m = sendKey(m, 'q')
		assert.NotContains(t, stripANSI(m.View().Content), "MIT License")
	})

	t.Run("fits a small screen", func(t *testing.T) {
		m := open(t, geom.Size{Width: 20, Height: 6})
		out := stripANSI(m.View().Content)
		for line := range strings.SplitSeq(out, "\n") {
			assert.LessOrEqual(t, len([]rune(line)), 20)
		}
	})

	t.Run("wheel does not scroll behind it", func(t *testing.T) {
		m := open(t, geom.Size{Width: 80, Height: 24})
		before := stripANSI(m.View().Content)

		m2, _ := m.Update(tea.MouseWheelMsg{
			X:      40,
			Y:      12,
			Button: tea.MouseWheelDown,
		})
		m = m2.(ui.Model)
		assert.Equal(t, before, stripANSI(m.View().Content))
	})

	t.Run("dismissed by a click", func(t *testing.T) {
		m := open(t, geom.Size{Width: 80, Height: 24})
		m2, _ := m.Update(tea.MouseClickMsg{
			X:      40,
			Y:      12,
			Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.NotContains(t, stripANSI(m.View().Content), "MIT License")
	})
}
