package view_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
)

func TestSession(t *testing.T) {
	t.Run("restores documents and layout", func(t *testing.T) {
		dir := t.TempDir()
		firstPath := filepath.Join(dir, "first.go")
		secondPath := filepath.Join(dir, "second.go")
		assert.NoError(t,
			os.WriteFile(firstPath, []byte("package main\n"), 0o644),
		)
		assert.NoError(t,
			os.WriteFile(secondPath, []byte("func main() {}\n"), 0o644),
		)

		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		first, err := e.OpenFile(firstPath)
		assert.NoError(t, err)
		first.SetMode(view.ModeSelect)
		first.SetOffset(view.Position{
			Anchor:           1,
			HorizontalOffset: 2,
			VerticalOffset:   3,
		})
		firstDoc, ok := e.Document(first.DocID())
		assert.True(t, ok)
		firstSel, err := core.NewSelection(
			[]core.Range{core.NewRange(1, 4)}, 0,
		)
		assert.NoError(t, err)
		firstDoc.SetSelectionFor(first.ID(), firstSel)

		secondDoc, err := e.SwitchOrOpenDoc(secondPath)
		assert.NoError(t, err)
		second, ok := e.HSplit(secondDoc.ID())
		assert.True(t, ok)
		second.SetMode(view.ModeInsert)
		secondSel := core.PointSelection(5)
		secondDoc.SetSelectionFor(second.ID(), secondSel)
		second.BeginFreeScroll(secondDoc.Revision(), secondSel)
		assert.NoError(t, e.SaveSession(
			sessionPath, map[string]string{"editor.cursorline": "true"},
		))
		data, err := os.ReadFile(sessionPath)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "[option]")
		assert.NotContains(t, string(data), "[[option]]")

		next := view.NewEditor(dir)
		values, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
		assert.Equal(t, "true", values["editor.cursorline"])

		views := next.AllViews()
		assert.Len(t, views, 2)
		assert.Equal(t, view.ModeSelect, views[0].Mode())
		assert.Equal(t, view.ModeInsert, views[1].Mode())
		assert.True(t, views[1].FreeScroll())
		assert.Equal(t, view.Position{
			Anchor:           1,
			HorizontalOffset: 2,
			VerticalOffset:   3,
		}, views[0].Offset())

		layout, ok := next.Tree().ContainerLayoutAt(views[0].ID())
		assert.True(t, ok)
		assert.Equal(t, view.LayoutHorizontal, layout)

		doc, ok := next.Document(views[0].DocID())
		assert.True(t, ok)
		assert.Equal(t,
			firstSel.Ranges(), doc.SelectionFor(views[0].ID()).Ranges(),
		)
		assert.Equal(t,
			firstSel.PrimaryIndex(),
			doc.SelectionFor(views[0].ID()).PrimaryIndex(),
		)
		doc, ok = next.Document(views[1].DocID())
		assert.True(t, ok)
		assert.Equal(t,
			secondSel.Ranges(), doc.SelectionFor(views[1].ID()).Ranges(),
		)
	})

	t.Run("missing file does not restore", func(t *testing.T) {
		dir := t.TempDir()
		e := view.NewEditor(dir)
		values, restored, err := e.RestoreSession(
			filepath.Join(dir, loader.WorkspaceDirName, view.SessionFile),
		)
		assert.NoError(t, err)
		assert.False(t, restored)
		assert.Nil(t, values)
	})

	t.Run("workspace session file path", func(t *testing.T) {
		dir := t.TempDir()
		path := view.WorkspaceSessionFile(dir)
		assert.Contains(t, path, view.SessionFile)
		assert.Contains(t, path, loader.WorkspaceDirName)
	})

	t.Run("single view no split", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		// write a single-view root (kind="view") that SaveSession never emits
		content := fmt.Sprintf(`version = 1

[[document]]
path = %q

[layout]
kind = "view"
document = 1
focused = true
mode = "NRM"
`, filePath)
		assert.NoError(t, os.MkdirAll(filepath.Dir(sessionPath), 0o755))
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte(content), 0o644),
		)
		next := view.NewEditor(dir)
		next.ResizeTree(80, 24)
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
		views := next.AllViews()
		assert.Len(t, views, 1)
		assert.Equal(t, view.ModeNormal, views[0].Mode())
	})

	t.Run("session outside workspace dir", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(dir, view.SessionFile)
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))
		next := view.NewEditor(dir)
		next.ResizeTree(80, 24)
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
	})

	t.Run("invalid TOML", func(t *testing.T) {
		dir := t.TempDir()
		sessionPath := filepath.Join(dir, view.SessionFile)
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte("{{{not toml"), 0o644),
		)
		e := view.NewEditor(dir)
		_, _, err := e.RestoreSession(sessionPath)
		assert.Error(t, err)
	})

	t.Run("unsupported version", func(t *testing.T) {
		dir := t.TempDir()
		sessionPath := filepath.Join(dir, view.SessionFile)
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte("version = 999\n"), 0o644),
		)
		e := view.NewEditor(dir)
		_, _, err := e.RestoreSession(sessionPath)
		assert.True(t, errors.Is(err, view.ErrSessionUnsupported))
	})

	t.Run("empty document list", func(t *testing.T) {
		dir := t.TempDir()
		sessionPath := filepath.Join(dir, view.SessionFile)
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte("version = 1\n"), 0o644),
		)
		e := view.NewEditor(dir)
		_, _, err := e.RestoreSession(sessionPath)
		assert.True(t, errors.Is(err, view.ErrSessionEmpty))
	})

	t.Run("invalid layout kind", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(dir, view.SessionFile)
		content := fmt.Sprintf(`version = 1

[[document]]
path = %q

[layout]
kind = "bogus"
`, filePath)
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte(content), 0o644),
		)
		e := view.NewEditor(dir)
		_, _, err := e.RestoreSession(sessionPath)
		assert.True(t, errors.Is(err, view.ErrSessionInvalid))
	})

	t.Run("view references missing document", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(dir, view.SessionFile)
		content := fmt.Sprintf(`version = 1

[[document]]
path = %q

[layout]
kind = "view"
document = 99
focused = true
`, filePath)
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte(content), 0o644),
		)
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, _, err := e.RestoreSession(sessionPath)
		assert.True(t, errors.Is(err, view.ErrSessionInvalid))
	})

	t.Run("corrupt selection primary index", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(dir, view.SessionFile)
		// primary=5 with one range: NewSelection fails,
		// falls back to PointSelection(0)
		content := fmt.Sprintf(`version = 1

[[document]]
path = %q

[layout]
kind = "view"
document = 1
focused = true

[layout.selection]
primary = 5

[[layout.selection.range]]
anchor = 0
head = 1
`, filePath)
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte(content), 0o644),
		)
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, restored, err := e.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
	})

	t.Run("restores scratch document", func(t *testing.T) {
		dir := t.TempDir()
		sessionPath := filepath.Join(dir, view.SessionFile)
		content := `version = 1

[[document]]
scratch = true
text = "scratch text\n"
language = "go"

[layout]
kind = "view"
document = 1
focused = true
`
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte(content), 0o644),
		)
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, restored, err := e.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "scratch text\n", doc.Text().String())
		assert.Equal(t, "go", doc.Lang())
	})

	t.Run("uses first view when none focused", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(dir, view.SessionFile)
		content := fmt.Sprintf(`version = 1

[[document]]
path = %q

[layout]
kind = "view"
document = 1
`, filePath)
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte(content), 0o644),
		)
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, restored, err := e.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
		assert.Len(t, e.AllViews(), 1)
	})

	t.Run("restores pane ratios after manual resize", func(t *testing.T) {
		dir := t.TempDir()
		leftPath := filepath.Join(dir, "left.go")
		rightPath := filepath.Join(dir, "right.go")
		assert.NoError(t,
			os.WriteFile(leftPath, []byte("package main\n"), 0o644))
		assert.NoError(t,
			os.WriteFile(rightPath, []byte("package main\n"), 0o644))
		sessionPath := filepath.Join(dir, view.SessionFile)

		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, err := e.OpenFile(leftPath)
		assert.NoError(t, err)
		rightDoc, err := e.SwitchOrOpenDoc(rightPath)
		assert.NoError(t, err)
		_, ok := e.VSplit(rightDoc.ID())
		assert.True(t, ok)

		// drag separator: left pane gets ~30 cols
		vs := e.Tree().Views()
		sepX := vs[0].View.Area().X + vs[0].View.Area().Width
		cID, idx, layout, ok := e.Tree().SeparatorAt(sepX, 0)
		assert.True(t, ok)
		e.Tree().MoveSeparator(cID, idx, layout, 30)

		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(80, 24)
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		views := next.AllViews()
		assert.Len(t, views, 2)
		// left pane width should be close to 30 (within 1 due to int rounding)
		assert.InDelta(t, 30, views[0].Area().Width, 1)
	})

	t.Run("restores registers", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644))
		sessionPath := filepath.Join(dir, view.SessionFile)

		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		e.Registers().Write('"', []string{"hello", "world"})
		e.Registers().Write('a', []string{"foo"})
		assert.NoError(t, e.SaveSession(sessionPath, nil))
		data, err := os.ReadFile(sessionPath)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "[register]")
		assert.NotContains(t, string(data), "[[register]]")

		next := view.NewEditor(dir)
		next.ResizeTree(80, 24)
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
		assert.Equal(t, []string{"hello", "world"}, next.Registers().Read('"'))
		assert.Equal(t, []string{"foo"}, next.Registers().Read('a'))
		assert.Nil(t, next.Registers().Read('z'))
	})

	t.Run("restores jump list", func(t *testing.T) {
		dir := t.TempDir()
		aPath := filepath.Join(dir, "a.go")
		bPath := filepath.Join(dir, "b.go")
		assert.NoError(t, os.WriteFile(aPath, []byte("package main\n"), 0o644))
		assert.NoError(t, os.WriteFile(bPath, []byte("package main\n"), 0o644))
		sessionPath := filepath.Join(dir, view.SessionFile)

		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		va, err := e.OpenFile(aPath)
		assert.NoError(t, err)
		docA, _ := e.Document(va.DocID())
		docB, err := e.SwitchOrOpenDoc(bPath)
		assert.NoError(t, err)

		va.PushJump(docA.ID(), 0, core.PointSelection(0))
		va.PushJump(docB.ID(), 5, core.PointSelection(5))
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(80, 24)
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		views := next.AllViews()
		assert.Len(t, views, 1)
		jumps := views[0].Jumps()
		assert.Len(t, jumps, 2)
		assert.Equal(t, 0, jumps[0].Anchor)
		assert.Equal(t, 5, jumps[1].Anchor)
	})

	t.Run("split child invalid", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(dir, view.SessionFile)
		content := fmt.Sprintf(`version = 1

[[document]]
path = %q

[layout]
kind = "split"
layout = "vertical"

[[layout.child]]
kind = "bogus"
`, filePath)
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte(content), 0o644),
		)
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, _, err := e.RestoreSession(sessionPath)
		assert.True(t, errors.Is(err, view.ErrSessionInvalid))
	})
}
