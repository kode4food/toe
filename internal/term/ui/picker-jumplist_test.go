package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/motion"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

const (
	jumplistAnchor            = 6
	jumplistPreviewAnchorLine = 1
	jumplistPreviewLineCount  = 70
	jumplistPreviewTargetLine = 45
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
		_ = sendSpecial(m, tea.KeyEnter)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		pos := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
		assert.Equal(t, anchor, pos)
	})

	t.Run("accept restores full selection", func(t *testing.T) {
		m, e, sel := jumplistSelectionModel(t)
		_ = sendSpecial(m, tea.KeyEnter)

		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got := doc.SelectionFor(v.ID())
		assert.Equal(t, sel.Ranges(), got.Ranges())
		assert.Equal(t, sel.PrimaryIndex(), got.PrimaryIndex())
	})

	t.Run("split accept restores full selection", func(t *testing.T) {
		m, e, sel := jumplistSelectionModel(t)
		before := len(e.AllViews())
		_ = sendModified(m, 's', tea.ModCtrl)

		assert.Equal(t, before+1, len(e.AllViews()))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got := doc.SelectionFor(v.ID())
		assert.Equal(t, sel.Ranges(), got.Ranges())
		assert.Equal(t, sel.PrimaryIndex(), got.PrimaryIndex())
	})

	t.Run("previews restored selection line", func(t *testing.T) {
		m := jumplistPreviewModel(t)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "TARGET_ONLY")
		assert.NotContains(t, out, "ANCHOR_ONLY")
	})
}

// jumplistModel opens a file, records a jump to the TARGET line, and returns a
// model with the jumplist picker open plus the editor and recorded anchor
func jumplistModel(t *testing.T) (ui.Model, *view.Editor, int) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	assert.NoError(t, os.WriteFile(path, []byte("l0\nl1\nTARGET\n"), 0o644))

	e := view.NewEditor(dir)
	v, err := e.OpenFile(path)
	assert.NoError(t, err)
	v.PushJump(v.DocID(), jumplistAnchor, core.PointSelection(jumplistAnchor))

	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "jumplist", m.PickerAction(motion.JumplistPicker),
		[]command.KeyEvent{char('j')},
	)
	m = resize(m, 120, 30)
	return sendKey(m, 'j'), e, jumplistAnchor
}

func jumplistSelectionModel(t *testing.T) (
	ui.Model, *view.Editor, core.Selection,
) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	assert.NoError(t, os.WriteFile(path, []byte("l0\nl1\nTARGET\n"), 0o644))

	e := view.NewEditor(dir)
	v, err := e.OpenFile(path)
	assert.NoError(t, err)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	sel, err := core.NewSelection([]core.Range{
		core.NewRange(0, 2),
		core.NewRange(jumplistAnchor, jumplistAnchor+6),
	}, 1)
	assert.NoError(t, err)
	v.PushJump(v.DocID(), sel.Primary().Cursor(doc.Text()), sel)

	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "jumplist", m.PickerAction(motion.JumplistPicker),
		[]command.KeyEvent{char('j')},
	)
	m = resize(m, 120, 30)
	return sendKey(m, 'j'), e, sel
}

func jumplistPreviewModel(t *testing.T) ui.Model {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	var b strings.Builder
	anchor := 0
	target := 0
	for i := range jumplistPreviewLineCount {
		switch i {
		case jumplistPreviewAnchorLine:
			anchor = b.Len()
			b.WriteString("ANCHOR_ONLY\n")
		case jumplistPreviewTargetLine:
			target = b.Len()
			b.WriteString("TARGET_ONLY\n")
		default:
			b.WriteString("padding\n")
		}
	}
	assert.NoError(t, os.WriteFile(path, []byte(b.String()), 0o644))

	e := view.NewEditor(dir)
	v, err := e.OpenFile(path)
	assert.NoError(t, err)
	v.PushJump(v.DocID(), anchor, core.PointSelection(target))

	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "jumplist", m.PickerAction(motion.JumplistPicker),
		[]command.KeyEvent{char('j')},
	)
	m = resize(m, 120, 12)
	return sendKey(m, 'j')
}
