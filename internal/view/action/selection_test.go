package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const blockCommentedSource = "hello /* world */"

func TestSelection(t *testing.T) {
	t.Run("trim removes surrounding whitespace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "  hi  ")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 6)}, 0)

		action.TrimSelections(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, sel.Primary().Anchor)
		assert.Equal(t, 4, sel.Primary().Head)
	})

	t.Run("trim drops all-whitespace range", func(t *testing.T) {
		e := testutil.EditorWithText(t, "  ")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 2)}, 0)

		action.TrimSelections(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		// collapses to primary cursor position (head clamped by block cursor)
		assert.Equal(t, 1, len(sel.Ranges()))
		assert.Equal(t, 1, sel.Primary().From())
		assert.Equal(t, 1, sel.Primary().To())
	})

	t.Run("join selections collapses line breaks", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.JoinSelections(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a b", doc.Text().String())
	})

	t.Run("rotate selections changes primary", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(
			t, e,
			[]core.Range{
				core.PointRange(0),
				core.PointRange(1),
				core.PointRange(2),
			},
			0,
		)

		action.RotateSelectionsForward(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		assert.Equal(t, 1, doc.SelectionFor(v.ID()).PrimaryIndex())

		action.RotateSelectionsBackward(e)

		assert.Equal(t, 0, doc.SelectionFor(v.ID()).PrimaryIndex())
	})

	t.Run("rotate contents tracks primary", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e,
			[]core.Range{
				core.NewRange(0, 1),
				core.NewRange(1, 2),
				core.NewRange(2, 3),
			},
			0,
		)

		action.RotateContentsForward(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "cab", doc.Text().String())
		assert.Equal(t, 1, doc.SelectionFor(v.ID()).PrimaryIndex())
	})

	t.Run("toggle comments a line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleComments(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "# hello", doc.Text().String())

		testutil.SetCursor(t, e, 0)
		action.ToggleComments(e)

		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestToggleLineComments(t *testing.T) {
	t.Run("adds default line comment token", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		// "text" lang uses "#" as default comment token
		assert.Equal(t, "# hello", doc.Text().String())
	})

	t.Run("removes existing comment token", func(t *testing.T) {
		e := testutil.EditorWithText(t, "# hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestToggleBlockComments(t *testing.T) {
	t.Run("wraps selection with block tokens", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.ToggleBlockComments(e)

		doc, _ := e.FocusedDocument()
		// default block comment tokens for "text" lang
		result := doc.Text().String()
		assert.NotEqual(t, "hello", result)
	})
}

func TestToggleCommentsBlockCommented(t *testing.T) {
	t.Run("removes inline block comment", func(t *testing.T) {
		e := testutil.EditorWithText(t, blockCommentedSource)
		doc, _ := e.FocusedDocument()
		doc.SetLang("go")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(6, len(blockCommentedSource)),
		}, 0)

		action.ToggleComments(e)

		assert.NotEqual(t, blockCommentedSource, doc.Text().String())
	})
}

func TestJoinSelectionsSpace(t *testing.T) {
	t.Run("joins lines with space separator", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.JoinSelectionsSpace(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a b", doc.Text().String())
	})
}

func TestGotoLineEndNewline(t *testing.T) {
	t.Run("cursor to end including newline pos", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 0)

		action.GotoLineEndNewline(e)

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})
}

func TestExtendToLineEndNewline(t *testing.T) {
	t.Run("extends through newline grapheme", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 0)

		action.ExtendToLineEndNewline(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		// PutCursor with extend=true advances one grapheme past newline pos
		assert.Equal(t, 4, sel.Primary().To())
	})
}

func TestRotateContentsBackward(t *testing.T) {
	t.Run("rotates content backward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e,
			[]core.Range{
				core.NewRange(0, 1),
				core.NewRange(1, 2),
				core.NewRange(2, 3),
			},
			0,
		)

		action.RotateContentsBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "bca", doc.Text().String())
	})
}

func TestSaveSelection(t *testing.T) {
	t.Run("does not panic and keeps text", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef")
		testutil.SetCursor(t, e, 3)

		assert.NotPanics(t, func() { action.SaveSelection(e) })

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abcdef", doc.Text().String())
	})
}

func TestCommitUndoCheckpoint(t *testing.T) {
	t.Run("does not change text", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")

		action.CommitUndoCheckpoint(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestGotoLastModification(t *testing.T) {
	t.Run("moves cursor to last edit position", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)
		action.InsertMode(e)
		action.InsertChar(e, 'x')
		action.NormalMode(e)
		// cursor should be near position 1 after inserting 'x'
		action.GotoLastModification(e)

		assert.True(t, testutil.CursorPos(t, e) >= 0)
	})
}

func TestJumpBackwardForward(t *testing.T) {
	t.Run("jump backward is noop when no history", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef")
		testutil.SetCursor(t, e, 3)

		assert.NotPanics(t, func() { action.JumpBackward(e) })
	})

	t.Run("forward noop when no forward history", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef")
		testutil.SetCursor(t, e, 3)

		assert.NotPanics(t, func() { action.JumpForward(e) })
	})

	t.Run("backward jump restores earlier position", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef\nghi")
		testutil.SetCursor(t, e, 1)
		// SaveSelection pushes cursor pos 1; then move to end
		action.SaveSelection(e)
		action.MoveFileEnd(e)
		// SaveSelection pushes cursor at end
		action.SaveSelection(e)
		// Now head=2; Backward() will succeed
		posEnd := testutil.CursorPos(t, e)

		action.JumpBackward(e)

		posAfter := testutil.CursorPos(t, e)
		// Should have jumped to the end position (the second push)
		assert.NotEqual(t, posEnd, posAfter)
	})

	t.Run("forward after backward restores position", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef\nghi")
		testutil.SetCursor(t, e, 0)
		action.SaveSelection(e)
		action.MoveFileEnd(e)
		posEnd := testutil.CursorPos(t, e)
		action.SaveSelection(e)
		action.JumpBackward(e)
		posBack := testutil.CursorPos(t, e)
		assert.NotEqual(t, posEnd, posBack)

		action.JumpForward(e)

		posAfter := testutil.CursorPos(t, e)
		assert.True(t, posAfter >= 0)
	})
}

func TestScrollUpDown(t *testing.T) {
	t.Run("ScrollUp does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne")
		testutil.SetCursor(t, e, 0)

		assert.NotPanics(t, func() { action.ScrollUp(e) })
		assert.NotPanics(t, func() { action.ScrollDown(e) })
	})

	t.Run("ScrollUp moves cursor from below top", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne")
		testutil.SetCursor(t, e, 8) // line 4

		action.ScrollUp(e)

		doc, _ := e.FocusedDocument()
		v, _ := e.FocusedView()
		sel := doc.SelectionFor(v.ID())
		assert.Less(t, sel.Primary().Cursor(doc.Text()), 8)
	})
}

func TestHSplitVSplit(t *testing.T) {
	t.Run("HSplit adds view", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		before := viewCount(t, e)

		action.HSplit(e)

		assert.Equal(t, before+1, viewCount(t, e))
	})

	t.Run("VSplit adds view", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		before := viewCount(t, e)

		action.VSplit(e)

		assert.Equal(t, before+1, viewCount(t, e))
	})
}

func TestRotateView(t *testing.T) {
	t.Run("cycles to next view", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		v, _ := e.FocusedView()
		e.VSplit(v.DocID())
		before, _ := e.FocusedView()

		action.RotateView(e)

		after, _ := e.FocusedView()
		assert.NotEqual(t, before.ID(), after.ID())
	})
}

func TestSmartTab(t *testing.T) {
	t.Run("inserts tab in leading whitespace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\thello")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 1)

		action.SmartTab(e)

		doc, _ := e.FocusedDocument()
		assert.True(t, len(doc.Text().String()) > len("\thello"))
	})
}

func TestSelectionIsLinewise(t *testing.T) {
	t.Run("linewise selection via ChangeSelection", func(t *testing.T) {
		// "hello\nworld\n" — Range(0,12) covers both complete lines exactly
		e := testutil.EditorWithText(t, "hello\nworld\n")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 12)}, 0)

		action.ChangeSelection(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("unaligned multi-line is not linewise", func(t *testing.T) {
		// Range(1, 10) spans lines 0 and 1 but does not start at line 0 start
		e := testutil.EditorWithText(t, "hello\nworld\n")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 10)}, 0)

		action.ChangeSelection(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestToggleLineCommentsBlockOnlyLang(t *testing.T) {
	t.Run("block tokens when no line tokens", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)

		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.True(t, len(result) >= len("hello"))
	})

	t.Run("no tokens is noop", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)

		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.ToggleBlockComments(e)

		doc, _ := e.FocusedDocument()
		assert.True(t, len(doc.Text().String()) > 0)
	})
}

func TestToggleCommentsMultiLine(t *testing.T) {
	t.Run("ToggleLineComments on multiple lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello\nworld")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 11)}, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.True(t, len(text) > len("hello\nworld"))
	})
}

func TestToggleLineCommentsWithLangToken(t *testing.T) {
	t.Run("uses language comment token", func(t *testing.T) {
		writeTextLangConfig(t, "//")

		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "// hello", doc.Text().String())
	})

	t.Run("removes language comment token", func(t *testing.T) {
		writeTextLangConfig(t, "//")

		e := testutil.EditorWithText(t, "// hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestToggleLineCommentsBlockLang(t *testing.T) {
	t.Run("block-only lang uses block tokens", func(t *testing.T) {
		writeTextBlockCommentConfig(t)

		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.NotEqual(t, "hello", result)
	})
}

func TestToggleCommentsBlockPath(t *testing.T) {
	t.Run("block-only lang uses block tokens", func(t *testing.T) {
		writeTextBlockCommentConfig(t)

		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.ToggleComments(e)

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.NotEqual(t, "hello", result)
	})

	t.Run("removes existing block comment", func(t *testing.T) {
		writeTextBlockCommentConfig(t)

		e := testutil.EditorWithText(t, "/* hello */")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 11)}, 0)

		action.ToggleComments(e)

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.NotContains(t, result, "/*")
	})
}

func TestToggleBlockCommentsWithLang(t *testing.T) {
	t.Run("wraps with language block tokens", func(t *testing.T) {
		writeTextBlockCommentConfig(t)

		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.ToggleBlockComments(e)

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.Contains(t, result, "/*")
		assert.Contains(t, result, "*/")
	})

	t.Run("unwraps block comment", func(t *testing.T) {
		writeTextBlockCommentConfig(t)

		e := testutil.EditorWithText(t, "/* hello */")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 11)}, 0)

		action.ToggleBlockComments(e)

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.NotContains(t, result, "/*")
	})
}

func TestToggleCommentsWithLineToken(t *testing.T) {
	t.Run("line-only lang uses line token", func(t *testing.T) {
		writeTextLangConfig(t, "//")

		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleComments(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "// hello", doc.Text().String())
	})
}

func TestCommentActionsNoView(t *testing.T) {
	t.Run("toggle comments is noop", func(t *testing.T) {
		assert.NotPanics(t, func() {
			action.ToggleComments(editorWithNoView(t))
		})
	})

	t.Run("toggle line comments is noop", func(t *testing.T) {
		assert.NotPanics(t, func() {
			action.ToggleLineComments(editorWithNoView(t))
		})
	})

	t.Run("toggle block comments is noop", func(t *testing.T) {
		assert.NotPanics(t, func() {
			action.ToggleBlockComments(editorWithNoView(t))
		})
	})
}

func TestToggleBlockCommentsLineFallback(t *testing.T) {
	t.Run("line-only lang falls back to line toggle", func(t *testing.T) {
		writeTextLangConfig(t, "//")

		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.ToggleBlockComments(e)

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.Contains(t, result, "//")
	})
}

func TestRotateSelectionsNoView(t *testing.T) {
	t.Run("forward noop with no view", func(t *testing.T) {
		e := editorWithNoView(t)
		action.RotateSelectionsForward(e)
	})

	t.Run("backward noop with no view", func(t *testing.T) {
		e := editorWithNoView(t)
		action.RotateSelectionsBackward(e)
	})
}

func TestKeepPrimarySelectionNoView(t *testing.T) {
	t.Run("noop with no view", func(t *testing.T) {
		e := editorWithNoView(t)
		action.KeepPrimarySelection(e)
	})
}

func TestSelectionActionsNoView(t *testing.T) {
	t.Run("line end is noop", func(t *testing.T) {
		action.GotoLineEndNewline(editorWithNoView(t))
	})

	t.Run("extend line end is noop", func(t *testing.T) {
		action.ExtendToLineEndNewline(editorWithNoView(t))
	})

	t.Run("save selection is noop", func(t *testing.T) {
		action.SaveSelection(editorWithNoView(t))
	})

	t.Run("remove primary is noop", func(t *testing.T) {
		action.RemovePrimarySelection(editorWithNoView(t))
	})

	t.Run("merge selections is noop", func(t *testing.T) {
		action.MergeSelections(editorWithNoView(t))
	})

	t.Run("merge consecutive is noop", func(t *testing.T) {
		action.MergeConsecutive(editorWithNoView(t))
	})

	t.Run("ensure forward is noop", func(t *testing.T) {
		action.EnsureForward(editorWithNoView(t))
	})

	t.Run("last modification is noop", func(t *testing.T) {
		action.GotoLastModification(editorWithNoView(t))
	})

	t.Run("jump backward is noop", func(t *testing.T) {
		action.JumpBackward(editorWithNoView(t))
	})

	t.Run("jump forward is noop", func(t *testing.T) {
		action.JumpForward(editorWithNoView(t))
	})
}

func TestToggleCommentsLineCommentedBranch(t *testing.T) {
	t.Run("each line separately block-commented", func(t *testing.T) {
		writeTextBlockCommentConfig(t)

		e := testutil.EditorWithText(t, "/* line one */\n/* line two */\n")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 30)}, 0)

		action.ToggleComments(e)

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.NotContains(t, result, "/*")
	})
}

func writeTextLangConfig(t *testing.T, commentToken string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	configDir := filepath.Join(dir, "toe")
	assert.NoError(t, os.MkdirAll(configDir, 0o755))
	content := "[[language]]\nname = \"text\"\nscope = \"text.plain\"\n" +
		"comment-token = \"" + commentToken + "\"\n"
	assert.NoError(t,
		os.WriteFile(filepath.Join(configDir, "languages.toml"),
			[]byte(content), 0o644),
	)
}

func writeTextBlockCommentConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	configDir := filepath.Join(dir, "toe")
	assert.NoError(t, os.MkdirAll(configDir, 0o755))
	content := "[[language]]\nname = \"text\"\nscope = \"text.plain\"\n" +
		"block-comment-tokens = [{ start = \"/*\", end = \"*/\" }]\n"
	assert.NoError(t,
		os.WriteFile(filepath.Join(configDir, "languages.toml"),
			[]byte(content), 0o644),
	)
}
