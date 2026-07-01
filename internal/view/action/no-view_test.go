package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestNoViewActions(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*view.Editor)
	}{
		{"align selections", action.AlignSelections},
		{"append mode", action.AppendMode},
		{"append line", action.AppendToLine},
		{"change selection", action.ChangeSelection},
		{"change no yank", action.ChangeSelectionNoYank},
		{"collapse selection", action.CollapseSelection},
		{"commit undo", action.CommitUndoCheckpoint},
		{"delete backward", action.DeleteCharBackward},
		{"delete forward", action.DeleteCharForward},
		{"delete selection", action.DeleteSelection},
		{"delete no yank", action.DeleteSelectionNoYank},
		{"delete word backward", action.DeleteWordBackward},
		{"delete word forward", action.DeleteWordForward},
		{"ensure forward", action.EnsureForward},
		{"exit select", action.ExitSelectMode},
		{"extend search next", func(e *view.Editor) {
			e.Registers().Write('/', []string{"a"})
			action.ExtendSearchNext(e)
		}},
		{"extend search prev", func(e *view.Editor) {
			e.Registers().Write('/', []string{"a"})
			action.ExtendSearchPrev(e)
		}},
		{"extend char left", action.ExtendCharLeft},
		{"extend char right", action.ExtendCharRight},
		{"extend file end", action.ExtendToFileEnd},
		{"extend file start", action.ExtendToFileStart},
		{"extend last line", action.ExtendToLastLine},
		{"extend line bounds", action.ExtendToLineBounds},
		{"extend line down", action.ExtendLineDown},
		{"extend line end", action.ExtendToLineEnd},
		{"extend line end newline", action.ExtendToLineEndNewline},
		{"extend line start", action.ExtendToLineStart},
		{"extend line up", action.ExtendLineUp},
		{"extend long word backward", action.ExtendPrevLongWordStart},
		{"extend long word end", action.ExtendNextLongWordEnd},
		{"extend long word forward", action.ExtendNextLongWordStart},
		{"extend prev long word end", action.ExtendPrevLongWordEnd},
		{"extend prev subword end", action.ExtendPrevSubWordEnd},
		{"extend prev word end", action.ExtendPrevWordEnd},
		{"extend subword end", action.ExtendNextSubWordEnd},
		{"extend subword forward", action.ExtendNextSubWordStart},
		{"extend subword start", action.ExtendPrevSubWordStart},
		{"extend word backward", action.ExtendPrevWordStart},
		{"extend word end", action.ExtendNextWordEnd},
		{"extend word forward", action.ExtendNextWordStart},
		{"flip selections", action.FlipSelections},
		{"goto line", func(e *view.Editor) { action.GotoLine(e, 1) }},
		{"goto line end newline", action.GotoLineEndNewline},
		{"goto last access", action.GotoLastAccessedFile},
		{"goto last modified", action.GotoLastModifiedFile},
		{"goto last modification", action.GotoLastModification},
		{"indent", action.Indent},
		{"insert char", func(e *view.Editor) { action.InsertChar(e, 'x') }},
		{"insert mode", action.InsertMode},
		{"insert newline", action.InsertNewline},
		{"insert line start", action.InsertAtLineStart},
		{"insert tab", action.InsertTab},
		{"jump backward", action.JumpBackward},
		{"jump forward", action.JumpForward},
		{"join", action.JoinSelections},
		{"join space", action.JoinSelectionsSpace},
		{"keep primary", action.KeepPrimarySelection},
		{"kill line end", action.KillToLineEnd},
		{"kill line start", action.KillToLineStart},
		{"line above", action.SelectLineAbove},
		{"line below", action.SelectLineBelow},
		{"copy next", action.CopyOnNextLine},
		{"copy prev", action.CopyOnPrevLine},
		{"match brackets", action.MatchBrackets},
		{"merge consecutive", action.MergeConsecutive},
		{"merge selections", action.MergeSelections},
		{"move down", action.MoveDown},
		{"move file end", action.MoveFileEnd},
		{"move file start", action.MoveFileStart},
		{"move left", action.MoveLeft},
		{"move line end", action.MoveLineEnd},
		{"move line non whitespace", action.MoveLineNonWhitespace},
		{"move line start", action.MoveLineStart},
		{"move long word backward", action.MoveLongWordBackward},
		{"move long word end", action.MoveLongWordEnd},
		{"move long word forward", action.MoveLongWordForward},
		{"move next subword end", action.MoveNextSubWordEnd},
		{"move next subword start", action.MoveNextSubWordStart},
		{"move prev long word end", action.MovePrevLongWordEnd},
		{"move prev subword end", action.MovePrevSubWordEnd},
		{"move prev subword start", action.MovePrevSubWordStart},
		{"move prev word end", action.MovePrevWordEnd},
		{"move right", action.MoveRight},
		{"move up", action.MoveUp},
		{"move word backward", action.MoveWordBackward},
		{"move word end", action.MoveWordEnd},
		{"move word forward", action.MoveWordForward},
		{"normal mode", action.NormalMode},
		{"open above", action.OpenAbove},
		{"open below", action.AddNewlineBelow},
		{"paste after", action.PasteAfter},
		{"paste before", action.PasteBefore},
		{"paste register", func(e *view.Editor) {
			action.PasteRegisterAtCursor(e, '"')
		}},
		{"clipboard replace", action.ClipboardReplace},
		{"remove primary", action.RemovePrimarySelection},
		{"repeat last motion", action.RepeatLastMotion},
		{"replace char", func(e *view.Editor) { action.ReplaceChar(e, 'x') }},
		{"replace yanked", action.ReplaceWithYanked},
		{"rotate contents backward", action.RotateContentsBackward},
		{"rotate contents forward", action.RotateContentsForward},
		{"rotate selections backward", action.RotateSelectionsBackward},
		{"rotate selections forward", action.RotateSelectionsForward},
		{"save selection", action.SaveSelection},
		{"search selection", action.SearchSelection},
		{"search selection word", action.SearchSelectionWord},
		{"search next", action.SearchNext},
		{"search prev", action.SearchPrev},
		{"select mode", action.SelectMode},
		{"scroll up", action.ScrollUp},
		{"scroll down", action.ScrollDown},
		{"page up", action.PageUp},
		{"page down", action.PageDown},
		{"half page up", action.HalfPageUp},
		{"half page down", action.HalfPageDown},
		{"cursor half up", action.PageCursorHalfUp},
		{"cursor half down", action.PageCursorHalfDown},
		{"align top", action.AlignViewTop},
		{"align center", action.AlignViewCenter},
		{"align bottom", action.AlignViewBottom},
		{"window top", action.GotoWindowTop},
		{"window center", action.GotoWindowCenter},
		{"window bottom", action.GotoWindowBottom},
		{"select all", action.SelectAll},
		{"shrink line bounds", action.ShrinkToLineBounds},
		{"smart tab", action.SmartTab},
		{"split newline", action.SplitSelectionOnNewline},
		{"surround add", func(e *view.Editor) { action.SurroundAdd(e, '(') }},
		{"surround delete", func(e *view.Editor) {
			action.SurroundDelete(e, '(')
		}},
		{"surround replace", func(e *view.Editor) {
			action.SurroundReplace(e, '(', '[')
		}},
		{"switch case", action.SwitchCase},
		{"text object around", func(e *view.Editor) {
			action.SelectTextObjectAround(e, 'w')
		}},
		{"text object inside", func(e *view.Editor) {
			action.SelectTextObjectInside(e, 'w')
		}},
		{"lowercase", action.SwitchToLowercase},
		{"uppercase", action.SwitchToUppercase},
		{"toggle block comments", action.ToggleBlockComments},
		{"toggle comments", action.ToggleComments},
		{"toggle line comments", action.ToggleLineComments},
		{"trim", action.TrimSelections},
		{"unindent", action.Unindent},
		{"increment", action.Increment},
		{"decrement", action.Decrement},
		{"yank", action.Yank},
		{"yank clipboard", action.YankToClipboard},
		{"yank main clipboard", action.YankMainToClipboard},
		{"yank primary", action.YankToPrimaryClipboard},
		{"yank join", func(e *view.Editor) { action.YankJoin(e, ",") }},
		{"reflow", func(e *view.Editor) { action.ReflowSelections(e, 80) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(editorWithNoView(t))
		})
	}
}

func TestNoViewSetLineEnding(t *testing.T) {
	t.Run("no view returns ErrNoView", func(t *testing.T) {
		e := editorWithNoView(t)

		err := action.SetLineEnding(e, core.LineEndingCRLF)

		assert.ErrorIs(t, err, view.ErrNoView)
	})
}

func TestNoViewReturnActions(t *testing.T) {
	t.Run("sort returns nil", func(t *testing.T) {
		err := action.SortSelections(editorWithNoView(t), false, false)
		assert.NoError(t, err)
	})

	t.Run("char info returns empty", func(t *testing.T) {
		assert.Empty(t, action.CharInfo(editorWithNoView(t)))
	})

	tests := []struct {
		name string
		fn   func(*view.Editor) error
	}{
		{"search forward", func(e *view.Editor) error {
			return action.SearchForward(e, "a")
		}},
		{"search backward", func(e *view.Editor) error {
			return action.SearchBackward(e, "a")
		}},
		{"select within regex", func(e *view.Editor) error {
			return action.SelectWithinRegex(e, "a")
		}},
		{"split by regex", func(e *view.Editor) error {
			return action.SplitSelectionByRegex(e, "a")
		}},
		{"keep matching", func(e *view.Editor) error {
			return action.KeepSelectionsMatching(e, "a")
		}},
		{"remove matching", func(e *view.Editor) error {
			return action.RemoveSelectionsMatching(e, "a")
		}},
		{"shell pipe", func(e *view.Editor) error {
			return action.ShellPipe(e, "cat")
		}},
		{"shell pipe to", func(e *view.Editor) error {
			return action.ShellPipeTo(e, "cat")
		}},
		{"shell keep pipe", func(e *view.Editor) error {
			return action.ShellKeepPipe(e, "cat")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, tc.fn(editorWithNoView(t)))
		})
	}
}
