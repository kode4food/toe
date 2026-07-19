package view_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
)

// fakePane stands in for a non-View pane (like a terminal) that persists
// itself as a reopenable session slot
type fakePane struct {
	id     view.Id
	editor *view.Editor
	area   geom.Area
	dirty  bool
}

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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
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

	t.Run("selection-only change saves in session", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		tx := core.NewTransaction(doc.Text()).
			WithSelection(core.PointSelection(3))
		assert.NoError(t, e.Apply(tx))
		assert.False(t, doc.Modified())

		assert.NoError(t, e.SaveSession(sessionPath, nil))
		next := view.NewEditor(dir)
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
		views := next.AllViews()
		assert.NotEmpty(t, views)
		nextDoc, ok := next.Document(views[0].DocID())
		assert.True(t, ok)
		assert.Equal(t, 3, nextDoc.Selection().Primary().Head)
	})

	t.Run("observers see restored documents", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		o := &recordingDocumentObserver{}
		next.AddDocumentObserver(o)
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)
		opened := make([]string, len(next.VisibleDocuments()))
		for i := range opened {
			opened[i] = "opened"
		}
		assert.Equal(t, opened, o.events)
	})

	t.Run("hidden buffer loads on access", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		otherPath := filepath.Join(dir, "other.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package lazy\n"), 0o644),
		)
		assert.NoError(t,
			os.WriteFile(otherPath, []byte("package other\n"), 0o644),
		)
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		_, err = e.OpenFile(otherPath) // hide file.go
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		var doc *view.Document
		for _, d := range next.AllDocuments() {
			if d.Path() == filePath {
				doc = d
			}
		}
		assert.NotNil(t, doc)
		assert.False(t, doc.Loaded())
		assert.Equal(t, "package lazy\n", doc.Text().String())
		assert.True(t, doc.Loaded())
	})

	t.Run("visible buffer loads on restore", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package eager\n"), 0o644),
		)
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		doc, ok := next.FocusedDocument()
		assert.True(t, ok)
		assert.True(t, doc.Loaded())
	})

	t.Run("hidden buffer keeps selection", func(t *testing.T) {
		dir := t.TempDir()
		aPath := filepath.Join(dir, "a.go")
		bPath := filepath.Join(dir, "b.go")
		assert.NoError(t, os.WriteFile(aPath, []byte("package aaaa\n"), 0o644))
		assert.NoError(t, os.WriteFile(bPath, []byte("package bbbb\n"), 0o644))
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		va, err := e.OpenFile(aPath)
		assert.NoError(t, err)
		aDoc, ok := e.Document(va.DocID())
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{core.NewRange(3, 7)}, 0)
		assert.NoError(t, err)
		aDoc.SetSelectionFor(va.ID(), sel)
		_, err = e.OpenFile(bPath) // hide a.go
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		var doc *view.Document
		for _, d := range next.AllDocuments() {
			if d.Path() == aPath {
				doc = d
			}
		}
		assert.NotNil(t, doc)
		assert.False(t, doc.Loaded())
		assert.Equal(t, sel.Ranges(), doc.Selection().Ranges())
	})

	t.Run("clamps selection to shrunk file", func(t *testing.T) {
		dir := t.TempDir()
		aPath := filepath.Join(dir, "a.go")
		bPath := filepath.Join(dir, "b.go")
		assert.NoError(t, os.WriteFile(aPath, []byte("package aaaa\n"), 0o644))
		assert.NoError(t, os.WriteFile(bPath, []byte("package bbbb\n"), 0o644))
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		va, err := e.OpenFile(aPath)
		assert.NoError(t, err)
		aDoc, ok := e.Document(va.DocID())
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{core.NewRange(8, 11)}, 0)
		assert.NoError(t, err)
		aDoc.SetSelectionFor(va.ID(), sel)
		_, err = e.OpenFile(bPath)
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		assert.NoError(t, os.WriteFile(aPath, []byte("hi\n"), 0o644)) // 3 chars

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		var doc *view.Document
		for _, d := range next.AllDocuments() {
			if d.Path() == aPath {
				doc = d
			}
		}
		assert.NotNil(t, doc)
		assert.Equal(t, "hi\n", doc.Text().String()) // triggers load + clamp
		primary := doc.Selection().Primary()
		assert.LessOrEqual(t, primary.Anchor, 3)
		assert.LessOrEqual(t, primary.Head, 3)
	})

	t.Run("missing file loads empty", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "gone.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package gone\n"), 0o644),
		)
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		assert.NoError(t, os.Remove(filePath))

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		var doc *view.Document
		for _, d := range next.AllDocuments() {
			if d.Path() == filePath {
				doc = d
			}
		}
		assert.NotNil(t, doc)
		assert.Equal(t, "", doc.Text().String())
		assert.True(t, doc.Loaded())
	})

	t.Run("drops unreadable restored file", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("permission checks don't apply when running as root")
		}
		dir := t.TempDir()
		subDir := filepath.Join(dir, "locked")
		filePath := filepath.Join(subDir, "file.go")
		otherPath := filepath.Join(dir, "other.go")
		assert.NoError(t, os.Mkdir(subDir, 0o755))
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package locked\n"), 0o644),
		)
		assert.NoError(t,
			os.WriteFile(otherPath, []byte("package other\n"), 0o644),
		)
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		_, err = e.OpenFile(otherPath)
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		assert.NoError(t, os.Chmod(subDir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(subDir, 0o755) })

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		for _, d := range next.AllDocuments() {
			assert.NotEqual(t, filePath, d.Path())
		}
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)
		assert.NoError(t, e.SaveSession(sessionPath, nil))
		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, err := e.OpenFile(leftPath)
		assert.NoError(t, err)
		rightDoc, err := e.SwitchOrOpenDoc(rightPath)
		assert.NoError(t, err)
		_, ok := e.VSplit(rightDoc.ID())
		assert.True(t, ok)

		// drag separator: left pane gets ~30 cols
		vs := e.Views()
		sepX := vs[0].View.Area().X + vs[0].View.Area().Width
		res, ok := e.Tree().SeparatorAt(geom.Point{X: sepX})
		assert.True(t, ok)
		e.Tree().MoveSeparator(
			res.ContainerID, res.ChildIdx, res.Layout, 30,
		)

		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		va, err := e.OpenFile(aPath)
		assert.NoError(t, err)
		docA, _ := e.Document(va.DocID())
		docB, err := e.SwitchOrOpenDoc(bPath)
		assert.NoError(t, err)

		va.PushJump(docA.ID(), 0, core.PointSelection(0))
		va.PushJump(docB.ID(), 5, core.PointSelection(5))
		assert.NoError(t, e.SaveSession(sessionPath, nil))

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, _, err := e.RestoreSession(sessionPath)
		assert.True(t, errors.Is(err, view.ErrSessionInvalid))
	})

	t.Run("round-trips a non-view pane's slot", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "file.go")
		assert.NoError(t,
			os.WriteFile(filePath, []byte("package main\n"), 0o644),
		)
		sessionPath := filepath.Join(
			dir, loader.WorkspaceDirName, view.SessionFile,
		)
		e := view.NewEditor(dir)
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, err := e.OpenFile(filePath)
		assert.NoError(t, err)

		e.Tree().Split(&fakePane{}, view.LayoutVertical)

		assert.NotPanics(t, func() {
			assert.NoError(t, e.SaveSession(sessionPath, nil))
		})

		next := view.NewEditor(dir)
		next.ResizeTree(geom.Size{Width: 80, Height: 24})
		// the pane rebuilds itself through its registered restorer, keyed by
		// the kind it persisted — no switch on pane type
		next.RegisterPaneRestorer(view.SessionKindTerminal,
			func(e *view.Editor, _ *view.PaneSession) (view.Pane, error) {
				return &fakePane{editor: e}, nil
			})
		_, restored, err := next.RestoreSession(sessionPath)
		assert.NoError(t, err)
		assert.True(t, restored)

		var restoredFake bool
		for _, p := range next.Tree().Traverse() {
			if _, ok := p.(*fakePane); ok {
				restoredFake = true
			}
		}
		assert.True(t, restoredFake)
	})
}

func (p *fakePane) ID() view.Id      { return p.id }
func (p *fakePane) SetID(id view.Id) { p.id = id }
func (p *fakePane) Split() (view.Pane, error) {
	return &fakePane{editor: p.editor}, nil
}
func (p *fakePane) Close() {
	if p.editor != nil {
		p.editor.RemovePane(p.id)
	}
}
func (p *fakePane) Discard()            {}
func (p *fakePane) Shutdown()           {}
func (p *fakePane) Area() geom.Area     { return p.area }
func (p *fakePane) SetArea(a geom.Area) { p.area = a }
func (p *fakePane) MarkDirty()          { p.dirty = true }

// ConsumeDirty reports whether MarkDirty was called since the last check
func (p *fakePane) ConsumeDirty() bool {
	d := p.dirty
	p.dirty = false
	return d
}

func (p *fakePane) SaveSession(w *view.SessionWriter) {
	w.SaveSlot(view.SessionKindTerminal, "")
}
func (p *fakePane) Path() string    { return "" }
func (p *fakePane) Mode() view.Mode { return view.ModeTerminal }
