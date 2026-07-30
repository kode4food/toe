package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

func TestTerminalLoading(t *testing.T) {
	e := editorWithText(t, "hello toe")
	tp, err := ui.NewTerminalPane(e, "cat", geom.Size{})
	assert.NoError(t, err)
	t.Cleanup(func() { _ = tp.Stop() })
	e.ReplacePane(e.Tree().Focus(), tp)
	m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

	starting := i18n.Text(i18n.StatusTerminalStarting)
	assert.Contains(t, stripANSI(m.View().Content), starting)

	tp.IngestOutput([]byte("ready"))
	assert.NotContains(t, stripANSI(m.View().Content), starting)
}

func TestScrollbackThemeBackground(t *testing.T) {
	t.Run("keeps the themed background", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := editorWithText(t, "hello toe")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		cont := m.TerminalAction()(e)
		assert.Nil(t, cont)
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
		assert.True(t, ok)
		t.Cleanup(func() { _ = tp.Stop() })

		waitForResize(t, tp)
		writeScrollbackLines(t, tp, 50)
		tp.ScrollLines(50)
		assert.Positive(t, tp.ScrollOffset())

		// mocha's ui.background is #1e1e2e (30,30,46)
		assert.Contains(t, m.View().Content, "\x1b[48;2;30;30;46m")
	})
}
