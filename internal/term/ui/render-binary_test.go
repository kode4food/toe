package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestBinaryPane(t *testing.T) {
	t.Run("opens read-only dump", func(t *testing.T) {
		root := t.TempDir()
		path := writeBinaryFile(t, filepath.Join(root, "bin", "tool"), 256)
		e := view.NewEditor(root)

		v, ok, err := ui.OpenPath(e, path, ui.PickerAcceptReplace)

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Nil(t, v)
		assert.Nil(t, e.FocusedDocument())
		pane, ok := e.FocusedPane().(*ui.BinaryPane)
		assert.True(t, ok)
		assert.Equal(t, view.ModeBinary, pane.Mode())

		m := resize(ui.New(e, command.NewKeymaps()), 80, 10)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "BIN")
		assert.Contains(t, out, filepath.Join("bin", "tool"))
		assert.Contains(t, out, "256 bytes")
		line := dumpLine(strings.Split(out, "\n"), "00000000")
		assert.Contains(t, line, "0e 0f")
		assert.Contains(t, line, "│................│")
		assert.NotContains(t, line, "|")
		raw := rawLineContaining(t, m.View().Content, "00000000")
		assert.Contains(t, raw,
			"\x1b[38;2;69;71;90m\x1b[48;2;30;30;46m00000000",
		)
		assert.Contains(t, raw, "\x1b[38;2;250;179;135m00 01")
		assert.Contains(t, raw, "\x1b[38;2;166;227;161m................")
	})

	t.Run("uses eight-byte width groups", func(t *testing.T) {
		root := t.TempDir()
		path := writeBinaryFile(t, filepath.Join(root, "tool"), 256)
		e := view.NewEditor(root)
		_, _, err := ui.OpenPath(e, path, ui.PickerAcceptReplace)
		assert.NoError(t, err)

		m := resize(ui.New(e, command.NewKeymaps()), 50, 10)
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		line := dumpLine(lines, "00000000")

		assert.Contains(t, line, "00 01 02 03 04 05 06 07")
		assert.NotContains(t, line, "08 09")
	})

	t.Run("dims when unfocused", func(t *testing.T) {
		root := t.TempDir()
		path := writeBinaryFile(t, filepath.Join(root, "tool"), 256)
		e := view.NewEditor(root)
		e.Options().InactiveDim = 50
		docID := e.Tree().Focus()
		m := resize(ui.New(e, command.NewKeymaps()), 100, 10)
		pane, err := ui.NewBinaryPane(e, path)
		assert.NoError(t, err)
		assert.True(t, e.SplitPane(pane, view.LayoutVertical))
		e.FocusPane(docID)

		raw := rawLineContaining(t, m.View().Content, "00000000")

		assert.Contains(t, raw, "\x1b[38;2;125;89;67m00 01")
	})

	t.Run("scrolls by rows", func(t *testing.T) {
		root := t.TempDir()
		path := writeBinaryFile(t, filepath.Join(root, "tool"), 512)
		e := view.NewEditor(root)
		_, _, err := ui.OpenPath(e, path, ui.PickerAcceptReplace)
		assert.NoError(t, err)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 6)

		m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

		assert.NotEmpty(t,
			dumpLine(
				strings.Split(stripANSI(m.View().Content), "\n"),
				"00000010",
			))
	})

	t.Run("restores path and offset", func(t *testing.T) {
		root := t.TempDir()
		path := writeBinaryFile(t, filepath.Join(root, "tool"), 512)
		session := filepath.Join(root, "session.toml")
		e := view.NewEditor(root)
		_, _, err := ui.OpenPath(e, path, ui.PickerAcceptReplace)
		assert.NoError(t, err)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 6)
		m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		assert.NoError(t, e.SaveSession(session, nil))

		next := view.NewEditor(root)
		nm := ui.New(next, command.NewKeymaps())
		_, restored, err := next.RestoreSession(session)
		assert.NoError(t, err)
		assert.True(t, restored)
		nm = resize(nm, 80, 6)

		_, ok := next.FocusedPane().(*ui.BinaryPane)
		assert.True(t, ok)
		assert.NotEmpty(t,
			dumpLine(
				strings.Split(stripANSI(nm.View().Content), "\n"),
				"00000010",
			))
	})
}

func writeBinaryFile(t *testing.T, path string, size int) string {
	t.Helper()
	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}
	assert.NoError(t, os.WriteFile(path, data, 0o755))
	return path
}

func dumpLine(lines []string, offset string) string {
	for _, line := range lines {
		if strings.Contains(line, offset+"  ") {
			return line
		}
	}
	return ""
}
