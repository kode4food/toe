package action_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type (
	focusedDocumentActionCase struct {
		name string
		fn   func(*view.Editor)
	}

	focusedDocumentErrorCase struct {
		name string
		fn   func(*view.Editor) error
		want error
	}
)

func TestFocusedDocumentGuardActions(t *testing.T) {
	tests := []focusedDocumentActionCase{
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
		{"flip selections", action.FlipSelections},
		{"goto file", func(e *view.Editor) { _, _ = action.GotoFile(e) }},
		{"goto line end newline", action.GotoLineEndNewline},
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
		{"move file end", action.MoveFileEnd},
		{"normal mode", action.NormalMode},
		{"open above", action.OpenAbove},
		{"open below", action.AddNewlineBelow},
		{"paste after", action.PasteAfter},
		{"paste before", action.PasteBefore},
		{"paste register", func(e *view.Editor) {
			action.PasteRegisterAtCursor(e, '"')
		}},
		{"clipboard replace", action.ClipboardReplace},
		{"reindent", action.ReindentSelections},
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
		{"select all", action.SelectAll},
		{"select mode", action.SelectMode},
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
			e := editorWithMissingFocusedDocument(t)

			tc.fn(e)

			_, ok := e.FocusedDocument()
			assert.False(t, ok)
		})
	}
}

func TestFocusedDocumentGuardErrors(t *testing.T) {
	path := ""
	tests := []focusedDocumentErrorCase{
		{"line ending", func(e *view.Editor) error {
			return action.SetLineEnding(e, core.LineEndingCRLF)
		}, view.ErrNoDocument},
		{"sort", func(e *view.Editor) error {
			return action.SortSelections(e, false, false)
		}, nil},
		{"search forward", func(e *view.Editor) error {
			return action.SearchForward(e, "a")
		}, nil},
		{"search backward", func(e *view.Editor) error {
			return action.SearchBackward(e, "a")
		}, nil},
		{"select within regex", func(e *view.Editor) error {
			return action.SelectWithinRegex(e, "a")
		}, nil},
		{"split by regex", func(e *view.Editor) error {
			return action.SplitSelectionByRegex(e, "a")
		}, nil},
		{"keep matching", func(e *view.Editor) error {
			return action.KeepSelectionsMatching(e, "a")
		}, nil},
		{"remove matching", func(e *view.Editor) error {
			return action.RemoveSelectionsMatching(e, "a")
		}, nil},
		{"shell pipe", func(e *view.Editor) error {
			return action.ShellPipe(e, "cat")
		}, nil},
		{"shell pipe to", func(e *view.Editor) error {
			return action.ShellPipeTo(e, "cat")
		}, nil},
		{"shell keep pipe", func(e *view.Editor) error {
			return action.ShellKeepPipe(e, "cat")
		}, nil},
		{"shell insert", func(e *view.Editor) error {
			return action.ShellInsertOutput(e, "printf x")
		}, nil},
		{"shell append", func(e *view.Editor) error {
			return action.ShellAppendOutput(e, "printf x")
		}, nil},
		{"read file", func(e *view.Editor) error {
			return action.ReadFile(e, path)
		}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := editorWithMissingFocusedDocument(t)
			if tc.name == "read file" {
				path = writeTempFile(t, "input.txt", "x")
			}

			err := tc.fn(e)

			if tc.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.want)
			}
		})
	}
}

func TestFocusedDocumentGuardStrings(t *testing.T) {
	assert.Empty(t, action.CharInfo(editorWithMissingFocusedDocument(t)))
}

func editorWithMissingFocusedDocument(t *testing.T) *view.Editor {
	t.Helper()
	e := editorWithText(t, "1\nabc\n")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	deleteDocument(e, v.DocID())
	_, ok = e.FocusedDocument()
	assert.False(t, ok)
	return e
}

func deleteDocument(e *view.Editor, did view.DocumentId) {
	f := reflect.ValueOf(e).Elem().FieldByName("docs")
	docs := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	docs.SetMapIndex(reflect.ValueOf(did), reflect.Value{})
}

func writeTempFile(t *testing.T, name, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	err := os.WriteFile(path, []byte(text), 0o644)
	assert.NoError(t, err)
	return path
}
