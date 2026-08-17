package ui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestToasts(t *testing.T) {
	t.Run("a message toasts bottom right", func(t *testing.T) {
		m := toastModel(t)

		m = m.ExecTypable("say")

		row := toastRow(m, "hello there")
		assert.NotEmpty(t, row)
		assert.Greater(t, strings.Index(row, "hello there"), 40)
		// the message sits inside a popup frame
		assert.Contains(t, row, promptBorderV)
		assert.NotEmpty(t, toastRow(m, "\u256d"))
		assert.NotEmpty(t, toastRow(m, "\u2570"))
	})

	t.Run("every queued message is kept", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		e.SetStatusMsg("first")
		e.SetStatusMsg("second")

		m = sendKey(m, 'x')

		content := stripANSI(m.View().Content)
		assert.Contains(t, content, "first")
		assert.Contains(t, content, "second")
	})

	t.Run("clears when the next command runs", func(t *testing.T) {
		m := toastModel(t)
		m = m.ExecTypable("say")
		assert.NotEmpty(t, toastRow(m, "hello there"))

		m = sendKey(m, 'x')

		assert.Empty(t, toastRow(m, "hello there"))
	})

	t.Run("stays clear of the statusline", func(t *testing.T) {
		m := toastModel(t)

		m = m.ExecTypable("say")

		lines := strings.Split(
			strings.TrimRight(stripANSI(m.View().Content), "\n"), "\n",
		)
		assert.NotContains(t, lines[len(lines)-1], "hello there")
	})

	t.Run("a click dismisses just that message", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		e.SetStatusMsg("first")
		e.SetStatusMsg("second")
		_ = m.View()

		at := toastPoint(t, m, "first")
		m = mouse(m, tea.MouseClickMsg{
			X: at.X, Y: at.Y, Button: tea.MouseLeft,
		})

		content := stripANSI(m.View().Content)
		assert.NotContains(t, content, "first")
		assert.Contains(t, content, "second")
	})

	t.Run("titled and logged to a buffer", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		e.SetStatusMsg("logged news")

		m = sendKey(m, 'x')

		assert.NotEmpty(t, toastRow(m, "Messages"))
		assert.Equal(t,
			"logged news\n", e.MessagesDocument().Text().String(),
		)
	})

	t.Run("errors outlast plain messages", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		assert.NoError(t, km.Register("boom", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Error: assert.AnError}
			},
			Modes:   view.ModeNormal,
			Aliases: []string{"boom"},
		}))
		m := resize(ui.New(e, km), 80, 24)

		m = m.ExecTypable("boom")

		assert.NotEmpty(t, toastRow(m, "assert.AnError"))
	})

	t.Run("severity picks the colour", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "mocha"
		km := command.NewKeymaps()
		assert.NoError(t, km.Register("boom", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Error: assert.AnError}
			},
			Modes:   view.ModeNormal,
			Aliases: []string{"boom"},
		}))
		m := resize(ui.New(e, km), 80, 24)
		e.SetStatusMsg("plain news")

		m = m.ExecTypable("boom")

		content := m.View().Content
		assert.NotEqual(t,
			foregrounds(rawToastRow(m, "plain news")),
			foregrounds(rawToastRow(m, "assert.AnError")),
		)
		assert.NotEmpty(t, content)
	})
}

func toastPoint(t *testing.T, m ui.Model, text string) geom.Point {
	t.Helper()
	for y, line := range strings.Split(stripANSI(m.View().Content), "\n") {
		if x := strings.Index(line, text); x >= 0 {
			return geom.Point{X: x, Y: y}
		}
	}
	t.Fatalf("toast %q not found", text)
	return geom.Point{}
}

func rawToastRow(m ui.Model, text string) string {
	for line := range strings.SplitSeq(m.View().Content, "\n") {
		if strings.Contains(stripANSI(line), text) {
			return line
		}
	}
	return ""
}

func toastModel(t *testing.T) ui.Model {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	assert.NoError(t, km.Register("say", command.Command{
		Run: func(*view.Editor, *command.Args) command.Result {
			return command.Result{Message: "hello there"}
		},
		Modes:   view.ModeNormal,
		Aliases: []string{"say"},
	}))
	assert.NoError(t, km.Register("quiet", command.Command{
		Run: func(*view.Editor, *command.Args) command.Result {
			return command.Result{}
		},
		Modes: view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('x')}},
		},
	}))
	return resize(ui.New(e, km), 80, 24)
}

func toastRow(m ui.Model, text string) string {
	for line := range strings.SplitSeq(stripANSI(m.View().Content), "\n") {
		if strings.Contains(line, text) {
			return line
		}
	}
	return ""
}
