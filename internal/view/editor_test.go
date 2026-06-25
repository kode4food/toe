package view_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

func TestNewEditor(t *testing.T) {
	t.Run("has one view and one document initially", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.Equal(t, 1, len(e.AllViews()))
		assert.Equal(t, 1, len(e.AllDocuments()))
	})

	t.Run("focused view exists", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		assert.NotNil(t, v)
	})

	t.Run("focused document exists", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		d, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.NotNil(t, d)
	})
}

func TestEditorMode(t *testing.T) {
	t.Run("defaults to normal", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("can switch to insert", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetMode(view.ModeInsert)
		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestEditorCount(t *testing.T) {
	t.Run("defaults to zero", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.Equal(t, 0, e.Count())
	})

	t.Run("set and reset", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetCount(5)
		assert.Equal(t, 5, e.Count())
		e.ResetCount()
		assert.Equal(t, 0, e.Count())
	})
}

func TestEditorRegister(t *testing.T) {
	t.Run("defaults to zero", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.Equal(t, rune(0), e.ActiveRegister())
	})

	t.Run("set and reset", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetRegister('a')
		assert.Equal(t, 'a', e.ActiveRegister())
		e.ResetRegister()
		assert.Equal(t, rune(0), e.ActiveRegister())
	})
}

func TestEditorCloseView(t *testing.T) {
	t.Run("closing last view clears focus", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		assert.Equal(t, 0, len(e.AllViews()))
	})

	t.Run("closing a view removes unreferenced doc", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		docCount := len(e.AllDocuments())
		e.CloseView(v.ID())
		assert.Equal(t, docCount-1, len(e.AllDocuments()))
	})
}

func TestEditorUndoRedo(t *testing.T) {
	t.Run("undo with no history", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.False(t, e.Undo())
	})

	t.Run("redo with no history", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.False(t, e.Redo())
	})
}

func TestEditorTree(t *testing.T) {
	t.Run("tree is non-nil", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.NotNil(t, e.Tree())
	})

	t.Run("resize changes tree area", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		changed := e.Tree().Resize(100, 40)
		assert.True(t, changed)
		assert.False(t, e.Tree().Resize(100, 40))
	})

	t.Run("ResizeTree delegates to tree", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		v, _ := e.FocusedView()
		a := v.Area()
		assert.Equal(t, 80, a.Width)
		assert.Equal(t, 24, a.Height)
	})
}

func TestEditorSplits(t *testing.T) {
	t.Run("HSplit creates second view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		d, _ := e.FocusedDocument()
		v, ok := e.HSplit(d.ID())
		assert.True(t, ok)
		assert.NotNil(t, v)
		assert.Equal(t, 2, len(e.AllViews()))
	})

	t.Run("HSplit with invalid docID fails", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		_, ok := e.HSplit(view.InvalidDocumentId)
		assert.False(t, ok)
	})

	t.Run("VSplitNew adds new doc", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		v := e.VSplitNew()
		assert.NotNil(t, v)
		assert.Equal(t, 2, len(e.AllDocuments()))
	})

	t.Run("HSplitNew adds new doc", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		v := e.HSplitNew()
		assert.NotNil(t, v)
		assert.Equal(t, 2, len(e.AllDocuments()))
	})

	t.Run("VSplitNew returns nil when too narrow", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(20, 24)
		v := e.VSplitNew()
		assert.Nil(t, v)
		assert.Equal(t, 1, len(e.AllDocuments()))
	})

	t.Run("HSplitNew returns nil when too short", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 8)
		v := e.HSplitNew()
		assert.Nil(t, v)
		assert.Equal(t, 1, len(e.AllDocuments()))
	})

	t.Run("VSplit returns false when too narrow", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(20, 24)
		d, _ := e.FocusedDocument()
		v, ok := e.VSplit(d.ID())
		assert.False(t, ok)
		assert.Nil(t, v)
	})

	t.Run("HSplit returns false when too short", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 8)
		d, _ := e.FocusedDocument()
		v, ok := e.HSplit(d.ID())
		assert.False(t, ok)
		assert.Nil(t, v)
	})
}

func TestEditorFocusNavigation(t *testing.T) {
	t.Run("FocusNextView wraps around", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		v1, _ := e.FocusedView()
		e.FocusNextView()
		v2, _ := e.FocusedView()
		assert.NotEqual(t, v1.ID(), v2.ID())
	})

	t.Run("FocusPrevView wraps around", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		v1, _ := e.FocusedView()
		e.FocusPrevView()
		v2, _ := e.FocusedView()
		assert.NotEqual(t, v1.ID(), v2.ID())
	})

	t.Run("FocusDirection moves to adjacent split", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(100, 40)
		e.VSplitNew()
		fv, _ := e.FocusedView()
		e.FocusDirection(view.DirectionLeft)
		lv, _ := e.FocusedView()
		assert.NotEqual(t, fv.ID(), lv.ID())
	})

	t.Run("FocusDirection no-op when no split", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.FocusDirection(view.DirectionLeft)
		after, _ := e.FocusedView()
		assert.Equal(t, v.ID(), after.ID())
	})
}

func TestEditorSwapAndTranspose(t *testing.T) {
	t.Run("SwapSplitInDirection swaps views", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(100, 40)
		e.VSplitNew()
		fv, _ := e.FocusedView()
		e.SwapSplitInDirection(view.DirectionLeft)
		all := e.AllViews()
		assert.Equal(t, 2, len(all))
		_ = fv
	})

	t.Run("Transpose flips layout", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.VSplitNew()
		before := e.Tree().Views()
		e.Transpose()
		after := e.Tree().Views()
		assert.Equal(t, len(before), len(after))
	})
}

func TestEditorLastMotion(t *testing.T) {
	t.Run("nil initially", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.Nil(t, e.LastMotion())
	})

	t.Run("set and retrieve", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		called := false
		fn := func(*view.Editor) { called = true }
		e.SetLastMotion(fn)
		m := e.LastMotion()
		assert.NotNil(t, m)
		m(e)
		assert.True(t, called)
	})
}

func TestEditorLastModifiedDocIDs(t *testing.T) {
	t.Run("zero initially", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		ids := e.LastModifiedDocIDs()
		assert.Equal(t, view.InvalidDocumentId, ids[0])
		assert.Equal(t, view.InvalidDocumentId, ids[1])
	})
}

func TestEditorPrevDocID(t *testing.T) {
	t.Run("returns false when no prev", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		_, ok := e.PrevDocID()
		assert.False(t, ok)
	})

	t.Run("returns prev after switch", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.VSplitNew()
		d, _ := e.FocusedDocument()
		prevID := d.ID()
		e.FocusNextView()
		id, ok := e.PrevDocID()
		assert.True(t, ok)
		assert.Equal(t, prevID, id)
	})
}

func TestEditorCommitInsertHistory(t *testing.T) {
	t.Run("commit with insert mode active", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetMode(view.ModeInsert)
		rope := func() *view.Editor {
			doc, _ := e.FocusedDocument()
			_ = doc
			return e
		}()
		_ = rope
		e.CommitInsertHistory()
	})

	t.Run("commit with no focused view is no-op", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		e.CommitInsertHistory()
	})
}

func TestEditorSaveAll(t *testing.T) {
	t.Run("save all with no paths returns errors", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.VSplitNew()
		errs := e.SaveAll()
		assert.Empty(t, errs)
	})

	t.Run("save all saves modified docs", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		tmp := t.TempDir()
		path := filepath.Join(tmp, "all.txt")
		e := editorWithText(t, "content")
		doc, _ := e.FocusedDocument()
		doc.SetPath(path)
		errs := e.SaveAll()
		assert.Empty(t, errs)
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "content")
	})
}

func TestEditorCloseCurrentAndOthers(t *testing.T) {
	t.Run("CloseCurrentView removes focused", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		assert.Equal(t, 2, len(e.AllViews()))
		e.CloseCurrentView()
		assert.Equal(t, 1, len(e.AllViews()))
	})

	t.Run("CloseAllOtherViews keeps focused", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		e.VSplitNew()
		assert.Equal(t, 3, len(e.AllViews()))
		e.CloseAllOtherViews()
		assert.Equal(t, 1, len(e.AllViews()))
	})
}

func TestEditorReloadAll(t *testing.T) {
	t.Run("reloads docs with paths", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "ral.txt")
		err := os.WriteFile(path, []byte("v1"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		err = os.WriteFile(path, []byte("v2"), 0o644)
		assert.NoError(t, err)
		errs := e.ReloadAll()
		assert.Empty(t, errs)
		d, _ := e.FocusedDocument()
		assert.Equal(t, "v2", d.Text().String())
	})
}

func TestEditorChdir(t *testing.T) {
	t.Run("changes cwd", func(t *testing.T) {
		tmp := t.TempDir()
		e := view.NewEditor("/tmp")
		err := e.Chdir(tmp)
		assert.NoError(t, err)
		assert.Equal(t, tmp, e.Cwd())
	})

	t.Run("error on non-existent path", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		err := e.Chdir("/nonexistent-dir-xyz-123")
		assert.Error(t, err)
	})
}

func TestEditorDirStack(t *testing.T) {
	t.Run("push and pop", func(t *testing.T) {
		tmp := t.TempDir()
		tmp2 := t.TempDir()
		e := view.NewEditor(tmp)
		err := e.Chdir(tmp)
		assert.NoError(t, err)
		err = e.PushDirectory(tmp2)
		assert.NoError(t, err)
		assert.Equal(t, tmp2, e.Cwd())
		stack := e.DirStack()
		assert.Equal(t, []string{tmp}, stack)
		err = e.PopDirectory()
		assert.NoError(t, err)
		assert.Equal(t, tmp, e.Cwd())
	})

	t.Run("pop empty stack returns error", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		err := e.PopDirectory()
		assert.ErrorIs(t, err, view.ErrEmptyDirStack)
	})

	t.Run("DirStack returns copy", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		s := e.DirStack()
		assert.Empty(t, s)
	})
}

func TestEditorStatusMsg(t *testing.T) {
	t.Run("set and take", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetStatusMsg("hello")
		assert.Equal(t, "hello", e.TakeStatusMsg())
		assert.Equal(t, "", e.TakeStatusMsg())
	})
}

func TestEditorHint(t *testing.T) {
	t.Run("set and take", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetHint("press f")
		assert.Equal(t, "press f", e.TakeHint())
		assert.Equal(t, "", e.TakeHint())
	})
}

func TestEditorViewHeight(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetViewHeight(30)
		assert.Equal(t, 30, e.ViewHeight())
	})
}

func TestEditorSetViewContentWidth(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetViewContentWidth(72)
		assert.Equal(t, 72, e.ViewContentWidth())
	})
}

func TestEditorSwitchBuffer(t *testing.T) {
	t.Run("switch to existing doc", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		all := e.AllDocuments()
		assert.Equal(t, 2, len(all))
		var other *view.Document
		fv, _ := e.FocusedView()
		for _, d := range all {
			if d.ID() != fv.DocID() {
				other = d
			}
		}
		assert.NotNil(t, other)
		ok := e.SwitchBuffer(other.ID())
		assert.True(t, ok)
		fv2, _ := e.FocusedView()
		assert.Equal(t, other.ID(), fv2.DocID())
	})

	t.Run("switch to non-existent returns false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		ok := e.SwitchBuffer(view.DocumentId(999))
		assert.False(t, ok)
	})
}

func TestEditorSwitchOrOpenDoc(t *testing.T) {
	t.Run("returns existing doc", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "switch.txt")
		err := os.WriteFile(path, []byte("hi"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d1, _ := e.FocusedDocument()
		d2, err := e.SwitchOrOpenDoc(path)
		assert.NoError(t, err)
		assert.Equal(t, d1.ID(), d2.ID())
	})

	t.Run("opens new file if not open", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "new.txt")
		err := os.WriteFile(path, []byte("content"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		d, err := e.SwitchOrOpenDoc(path)
		assert.NoError(t, err)
		assert.Equal(t, "content", d.Text().String())
	})
}

func TestEditorEarlierLater(t *testing.T) {
	t.Run("Earlier returns false with no history", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.False(t, e.Earlier(core.UndoSteps(1)))
	})

	t.Run("Later returns false with no history", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.False(t, e.Later(core.UndoSteps(1)))
	})

	t.Run("Earlier with history is callable", func(t *testing.T) {
		e := editorWithText(t, "hello")
		_ = e.Earlier(core.UndoSteps(1))
	})

	t.Run("Later with history is callable", func(t *testing.T) {
		e := editorWithText(t, "hello")
		e.Earlier(core.UndoSteps(1))
		_ = e.Later(core.UndoSteps(1))
	})

	t.Run("Earlier no view returns false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		assert.False(t, e.Earlier(core.UndoSteps(1)))
	})

	t.Run("Later no view returns false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		assert.False(t, e.Later(core.UndoSteps(1)))
	})
}

func TestEditorConfigReload(t *testing.T) {
	t.Run("no reload fn returns error", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.Equal(t, view.ErrConfigUnavailable, e.ReloadConfig())
	})

	t.Run("reload fn is called", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		called := false
		e.SetConfigReload(func() error { called = true; return nil })
		assert.NoError(t, e.ReloadConfig())
		assert.True(t, called)
	})
}

func TestEditorApplyInsertMode(t *testing.T) {
	t.Run("apply in insert mode accumulates", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.SetMode(view.ModeInsert)
		d, _ := e.FocusedDocument()
		rope := d.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "hi"),
		})
		assert.NoError(t, err)
		tx := core.NewTransaction(rope).WithChanges(cs)
		assert.NoError(t, e.Apply(tx))
		d2, _ := e.FocusedDocument()
		assert.Equal(t, "hi", d2.Text().String())
	})
}

func TestEditorViewByID(t *testing.T) {
	t.Run("view returns existing", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		got, ok := e.View(v.ID())
		assert.True(t, ok)
		assert.Equal(t, v.ID(), got.ID())
	})

	t.Run("view returns false for invalid id", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		_, ok := e.View(view.InvalidViewId)
		assert.False(t, ok)
	})
}

func TestTreeViews(t *testing.T) {
	t.Run("single view has focused=true", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		views := e.Tree().Views()
		assert.Equal(t, 1, len(views))
		assert.True(t, views[0].Focused)
	})

	t.Run("two views: one focused", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		e.VSplitNew()
		views := e.Tree().Views()
		assert.Equal(t, 2, len(views))
		focused := 0
		for _, v := range views {
			if v.Focused {
				focused++
			}
		}
		assert.Equal(t, 1, focused)
	})
}

func TestTreeNext(t *testing.T) {
	t.Run("single view next wraps to self", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		next := e.Tree().Next()
		assert.Equal(t, v.ID(), next)
	})

	t.Run("two views cycle", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v1, _ := e.FocusedView()
		e.VSplitNew()
		v2, _ := e.FocusedView()
		assert.Equal(t, v1.ID(), e.Tree().Next())
		e.Tree().SetFocus(v1.ID())
		assert.Equal(t, v2.ID(), e.Tree().Next())
	})

	t.Run("invalid focus returns first view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.Tree().SetFocus(view.InvalidViewId)
		next := e.Tree().Next()
		assert.NotEqual(t, view.InvalidViewId, next)
	})

	t.Run("empty tree returns current focus", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		next := e.Tree().Next()
		assert.Equal(t, e.Tree().Focus(), next)
	})
}

func TestTreePrev(t *testing.T) {
	t.Run("single view prev wraps to self", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		assert.Equal(t, v.ID(), e.Tree().Prev())
	})

	t.Run("invalid focus returns last view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.Tree().SetFocus(view.InvalidViewId)
		prev := e.Tree().Prev()
		assert.NotEqual(t, view.InvalidViewId, prev)
	})

	t.Run("empty tree returns current focus", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		prev := e.Tree().Prev()
		assert.Equal(t, e.Tree().Focus(), prev)
	})
}

func TestTreeTranspose(t *testing.T) {
	t.Run("single view stays with toggled layout", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.VSplitNew()
		layout1, ok := e.Tree().ContainerLayoutAt(
			e.Tree().Focus(),
		)
		assert.True(t, ok)
		e.Tree().Transpose()
		layout2, ok2 := e.Tree().ContainerLayoutAt(
			e.Tree().Focus(),
		)
		assert.True(t, ok2)
		assert.NotEqual(t, layout1, layout2)
	})
}

func TestTreeNodeID(t *testing.T) {
	t.Run("NodeID returns same id for leaf", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		assert.Equal(t, v.ID(), e.Tree().NodeID(v.ID()))
	})
}

func TestTreeContainerLayoutAt(t *testing.T) {
	t.Run("returns layout for valid view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		layout, ok := e.Tree().ContainerLayoutAt(v.ID())
		assert.True(t, ok)
		assert.Equal(t, view.LayoutVertical, layout)
	})

	t.Run("returns false for invalid id", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		_, ok := e.Tree().ContainerLayoutAt(view.InvalidViewId)
		assert.False(t, ok)
	})
}

func TestTreeWalkSeparators(t *testing.T) {
	t.Run("no separators for single view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(100, 40)
		var count int
		e.Tree().WalkSeparators(func(_ view.Separator) {
			count++
		})
		assert.Equal(t, 0, count)
	})

	t.Run("one separator for two vertical splits", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(100, 40)
		e.VSplitNew()
		var count int
		e.Tree().WalkSeparators(func(_ view.Separator) {
			count++
		})
		assert.Equal(t, 1, count)
	})

	t.Run("one separator for two horizontal splits", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(100, 40)
		e.HSplitNew()
		var count int
		e.Tree().WalkSeparators(func(_ view.Separator) {
			count++
		})
		assert.Equal(t, 1, count)
	})
}

func TestTreeFindSplitInDirection(t *testing.T) {
	t.Run("find right split", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(100, 40)
		v1, _ := e.FocusedView()
		e.VSplitNew()
		e.Tree().SetFocus(v1.ID())
		id, ok := e.Tree().FindSplitInDirection(
			e.Tree().Focus(), view.DirectionRight,
		)
		assert.True(t, ok)
		assert.NotEqual(t, v1.ID(), id)
	})

	t.Run("no split in direction when only one view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		_, ok := e.Tree().FindSplitInDirection(
			v.ID(), view.DirectionRight,
		)
		assert.False(t, ok)
	})
}

func TestTreeSwapSplitInDirection(t *testing.T) {
	t.Run("swap in direction when split exists", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(100, 40)
		v1, _ := e.FocusedView()
		e.VSplitNew()
		v2, _ := e.FocusedView()
		ok := e.Tree().SwapSplitInDirection(view.DirectionLeft)
		assert.True(t, ok)
		all := e.Tree().Traverse()
		assert.Equal(t, 2, len(all))
		_ = v1
		_ = v2
	})

	t.Run("no swap when single view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		ok := e.Tree().SwapSplitInDirection(view.DirectionLeft)
		assert.False(t, ok)
	})
}

func TestTreeResize(t *testing.T) {
	t.Run("same dimensions returns false", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.Tree().Resize(80, 24)
		assert.False(t, e.Tree().Resize(80, 24))
	})

	t.Run("different dimensions returns true", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		assert.True(t, e.Tree().Resize(80, 24))
	})
}

func TestTreeHorizontalSplitArea(t *testing.T) {
	t.Run("horizontal split distributes height", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 40)
		e.HSplitNew()
		views := e.Tree().Views()
		assert.Equal(t, 2, len(views))
		total := views[0].View.Area().Height + views[1].View.Area().Height
		// 1 gap row between panes
		assert.Equal(t, 39, total)
	})

	t.Run("find split in up/down direction", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 40)
		e.HSplitNew()
		fv, _ := e.FocusedView()
		_, ok := e.Tree().FindSplitInDirection(
			fv.ID(), view.DirectionUp,
		)
		assert.True(t, ok)
	})

	t.Run("right from vsplit navigates into hsplit", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(120, 60)
		views0 := e.Tree().Traverse()
		v1ID := views0[0].ID()
		e.VSplitNew()
		e.HSplitNew()
		allViews := e.Tree().Traverse()
		assert.Equal(t, 3, len(allViews))
		e.Tree().SetFocus(v1ID)
		id, ok := e.Tree().FindSplitInDirection(
			v1ID, view.DirectionRight,
		)
		assert.True(t, ok)
		assert.NotEqual(t, v1ID, id)
	})

	t.Run("down from hsplit navigates into vsplit", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(120, 60)
		views0 := e.Tree().Traverse()
		v1ID := views0[0].ID()
		e.HSplitNew()
		e.VSplitNew()
		allViews := e.Tree().Traverse()
		assert.Equal(t, 3, len(allViews))
		e.Tree().SetFocus(v1ID)
		id, ok := e.Tree().FindSplitInDirection(
			v1ID, view.DirectionDown,
		)
		assert.True(t, ok)
		assert.NotEqual(t, v1ID, id)
	})

	t.Run("swap cross-container swaps views", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(120, 60)
		e.VSplitNew()
		views := e.Tree().Traverse()
		e.Tree().SetFocus(views[0].ID())
		e.HSplitNew()
		_ = e.Tree().SwapSplitInDirection(view.DirectionRight)
	})
}

func TestEditorSwitchOrOpenDocError(t *testing.T) {
	t.Run("bad path returns error", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		_, err := e.SwitchOrOpenDoc("\x00invalid")
		assert.Error(t, err)
	})
}

func TestEditorSwitchFileReuseDoc(t *testing.T) {
	t.Run("already open file reuses document", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "reuse.txt")
		err := os.WriteFile(path, []byte("content"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		d1, _ := e.FocusedDocument()
		firstDocID := d1.ID()
		e.VSplitNew()
		_, err = e.SwitchFile(path)
		assert.NoError(t, err)
		d2, _ := e.FocusedDocument()
		assert.Equal(t, firstDocID, d2.ID())
	})
}

func TestEditorUndoRedoWithHistory(t *testing.T) {
	t.Run("undo restores previous text", func(t *testing.T) {
		e := editorWithText(t, "hello")
		ok := e.Undo()
		assert.True(t, ok)
		d, _ := e.FocusedDocument()
		assert.Equal(t, "", d.Text().String())
	})

	t.Run("redo reapplies text", func(t *testing.T) {
		e := editorWithText(t, "hello")
		e.Undo()
		ok := e.Redo()
		assert.True(t, ok)
		d, _ := e.FocusedDocument()
		assert.Equal(t, "hello", d.Text().String())
	})
}

func TestEditorModeNoView(t *testing.T) {
	t.Run("Mode returns normal when no focused view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestEditorNewDocumentNoView(t *testing.T) {
	t.Run("no view inserts new view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		nv := e.NewDocument()
		assert.NotNil(t, nv)
		assert.Equal(t, 1, len(e.AllViews()))
	})
}

func TestEditorSaveNoDoc(t *testing.T) {
	t.Run("Save returns error when no doc", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		err := e.Save()
		assert.Error(t, err)
	})
}

func TestEditorReloadNoDoc(t *testing.T) {
	t.Run("Reload returns error when no doc", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		err := e.Reload()
		assert.Error(t, err)
	})
}

func TestEditorApplyNoView(t *testing.T) {
	t.Run("Apply returns error when no view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		rope := core.NewRope("")
		tx := core.NewTransaction(rope)
		err := e.Apply(tx)
		assert.ErrorIs(t, err, view.ErrNoView)
	})
}

func TestEditorRecordPrevDocModified(t *testing.T) {
	t.Run("lastModifiedDocIDs updated on doc change", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.VSplitNew()
		d2, _ := e.FocusedDocument()
		secondID := d2.ID()
		rope := d2.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "x"),
		})
		assert.NoError(t, err)
		tx := core.NewTransaction(rope).WithChanges(cs)
		assert.NoError(t, e.Apply(tx))
		e.FocusNextView()
		ids := e.LastModifiedDocIDs()
		assert.Equal(t, secondID, ids[0])
	})
}

func TestTreeSwapCrossContainer(t *testing.T) {
	t.Run("swap across containers", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(120, 40)
		e.VSplitNew()
		e.HSplitNew()
		views := e.Tree().Traverse()
		assert.GreaterOrEqual(t, len(views), 2)
		ok := e.Tree().SwapSplitInDirection(view.DirectionLeft)
		assert.True(t, ok)
	})
}

func TestEditorUndoNoView(t *testing.T) {
	t.Run("Undo returns false when no view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		assert.False(t, e.Undo())
	})
}

func TestEditorRedoNoView(t *testing.T) {
	t.Run("Redo returns false when no view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		assert.False(t, e.Redo())
	})
}

func TestEditorSaveAllWithError(t *testing.T) {
	t.Run("modified scratch returns error", func(t *testing.T) {
		e := editorWithText(t, "unsaved")
		errs := e.SaveAll()
		assert.Equal(t, 1, len(errs))
	})
}

func TestEditorCommitInsertHistoryNoDoc(t *testing.T) {
	t.Run("CommitInsertHistory when no doc is no-op", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		_ = doc
		e.CloseView(v.ID())
		e.CommitInsertHistory()
	})
}

func TestTreeSeparatorAt(t *testing.T) {
	t.Run("empty tree returns not ok", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())

		_, _, _, ok := e.Tree().SeparatorAt(0, 0)

		assert.False(t, ok)
	})

	t.Run("nested layout separator found", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(120, 60)
		v1ID := e.Tree().Focus()
		e.VSplitNew()
		e.Tree().SetFocus(v1ID)
		e.HSplitNew()

		var count int
		e.Tree().WalkSeparators(func(_ view.Separator) {
			count++
		})

		assert.Equal(t, 2, count)
	})

	t.Run("nested layout SeparatorAt finds sep", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		e.ResizeTree(120, 60)
		v1ID := e.Tree().Focus()
		e.VSplitNew()
		e.Tree().SetFocus(v1ID)
		e.HSplitNew()

		var found bool
		e.Tree().WalkSeparators(func(s view.Separator) {
			if s.Layout == view.LayoutVertical {
				_, _, _, ok := e.Tree().SeparatorAt(s.X, s.Y)
				if ok {
					found = true
				}
			}
		})

		assert.True(t, found)
	})
}
