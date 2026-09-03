package ui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
)

func TestOptionCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: each subtest shells out to a real git repo")
	}
	testutil.RequireGit(t)

	t.Run("tab moves the focus", func(t *testing.T) {
		repo, m := discardCapture(t)

		m = updateAndFeed(m, tea.KeyPressMsg{Code: tea.KeyTab})
		assert.Equal(t, "48;2;147;153;178", focusedButtonBg(m, 'Y'))

		m = updateAndFeed(m, tea.KeyPressMsg{
			Code: tea.KeyTab, Mod: tea.ModShift,
		})
		assert.Equal(t, "48;2;147;153;178", focusedButtonBg(m, 'N'))
		assert.Equal(t, " M mod.txt", gitStatus(t, repo))
	})

	t.Run("l and h move the focus", func(t *testing.T) {
		_, m := discardCapture(t)

		m = sendKeyAndFeed(m, 'l')
		assert.Equal(t, "48;2;147;153;178", focusedButtonBg(m, 'Y'))

		m = sendKeyAndFeed(m, 'h')
		assert.Equal(t, "48;2;147;153;178", focusedButtonBg(m, 'N'))
	})

	t.Run("focus wraps around the row", func(t *testing.T) {
		_, m := discardCapture(t)

		// no is the last button, so moving on lands back on the first
		m = sendKeyAndFeed(m, 'l')
		m = sendKeyAndFeed(m, 'l')
		assert.Equal(t, "48;2;147;153;178", focusedButtonBg(m, 'N'))
	})

	t.Run("space takes the focused answer", func(t *testing.T) {
		repo, m := discardCapture(t)

		m = sendKeyAndFeed(sendKeyAndFeed(m, 'l'), ' ')

		assert.NotContains(t, stripANSI(m.View().Content), "Discard changes")
		assert.Empty(t, gitStatus(t, repo))
	})

	t.Run("escape dismisses unanswered", func(t *testing.T) {
		repo, m := discardCapture(t)

		m = updateAndFeed(m, tea.KeyPressMsg{Code: tea.KeyEscape})

		assert.NotContains(t, stripANSI(m.View().Content), "Discard changes")
		assert.Equal(t, " M mod.txt", gitStatus(t, repo))
	})

	t.Run("an unbound key dismisses unanswered", func(t *testing.T) {
		repo, m := discardCapture(t)

		m = sendKeyAndFeed(m, 'q')

		assert.NotContains(t, stripANSI(m.View().Content), "Discard changes")
		assert.Equal(t, " M mod.txt", gitStatus(t, repo))
	})

	t.Run("a click dismisses unanswered", func(t *testing.T) {
		repo, m := discardCapture(t)

		at := popupCell(m, "Discard changes")
		m = updateAndFeed(m, tea.MouseClickMsg{
			X: at.X, Y: at.Y, Button: tea.MouseLeft,
		})

		assert.NotContains(t, stripANSI(m.View().Content), "Discard changes")
		assert.Equal(t, " M mod.txt", gitStatus(t, repo))
	})

	t.Run("the wheel leaves the popup up", func(t *testing.T) {
		_, m := discardCapture(t)
		before := m.View().Content

		m = updateAndFeed(m, tea.MouseWheelMsg{
			X: 60, Y: 10, Button: tea.MouseWheelDown,
		})

		// the wheel is swallowed, so neither the popup nor the picker under
		// it moves
		assert.Equal(t, before, m.View().Content)
	})

	t.Run("a resize leaves the popup up", func(t *testing.T) {
		_, m := discardCapture(t)

		m = updateAndFeed(m, tea.WindowSizeMsg{Width: 100, Height: 30})

		// a message the popup has no answer for falls through to the layers
		// below, which re-lay it out rather than closing it
		assert.Contains(t, stripANSI(m.View().Content), "Discard changes")
	})

	t.Run("a long question wraps inside the popup", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		name := strings.Repeat("long-name-", 6) + ".txt"
		path := testutil.GitCommitFile(t, repo, name, "one\n")
		testutil.WriteFile(t, path, "two\n")

		m := sendCtrl(changedFilePicker(t, repo), 'r')

		// the question outgrows the popup's width cap, so it takes two rows
		// and the buttons still sit under it
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "Discard changes to")
		assert.Contains(t, out, name[:20])
		assert.Contains(t, stripANSI(popupRow(m, "y Yes")), "n No")
	})

	t.Run("a danger capture tints its ground", func(t *testing.T) {
		_, m := discardCapture(t)

		// the popup ground is tinted toward the error color, so it differs
		// from the picker's own popup background behind it
		question := styledRuneStyles(popupRow(m, "Discard changes"))
		picker := styledRuneStyles(popupRow(m, "Changed Files"))
		assert.Equal(t, "48;2;87;67;88", question['D'].bg)
		assert.Equal(t, "48;2;49;50;68", picker['C'].bg)
	})
}

func discardCapture(t *testing.T) (string, ui.Model) {
	t.Helper()
	repo := testutil.GitRepo(t)
	path := testutil.GitCommitFile(t, repo, "mod.txt", "one\n")
	testutil.WriteFile(t, path, "two\n")
	m := sendCtrl(changedFilePicker(t, repo), 'r')
	assert.Contains(t, stripANSI(m.View().Content), "Discard changes")
	return repo, m
}

func popupCell(m ui.Model, text string) geom.Point {
	for y, row := range strings.Split(m.View().Content, "\n") {
		if x := strings.Index(stripANSI(row), text); x >= 0 {
			return geom.Point{X: x, Y: y}
		}
	}
	return geom.Point{X: -1, Y: -1}
}

func focusedButtonBg(m ui.Model, label rune) string {
	return styledRuneStyles(popupRow(m, "y Yes"))[label].bg
}
