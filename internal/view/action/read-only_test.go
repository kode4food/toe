package action_test

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestReadOnlyActions(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*view.Editor)
	}{
		{"align selections", action.AlignSelections},
		{"change selection", action.ChangeSelection},
		{"change no yank", action.ChangeSelectionNoYank},
		{"delete backward", action.DeleteCharBackward},
		{"delete forward", action.DeleteCharForward},
		{"delete selection", action.DeleteSelection},
		{"delete no yank", action.DeleteSelectionNoYank},
		{"delete word backward", action.DeleteWordBackward},
		{"delete word forward", action.DeleteWordForward},
		{"increment", action.Increment},
		{"decrement", action.Decrement},
		{"indent", action.Indent},
		{"insert char", func(e *view.Editor) { action.InsertChar(e, 'x') }},
		{"insert newline", action.InsertNewline},
		{"insert tab", action.InsertTab},
		{"join", action.JoinSelections},
		{"join space", action.JoinSelectionsSpace},
		{"kill line end", action.KillToLineEnd},
		{"kill line start", action.KillToLineStart},
		{"open above", action.OpenAbove},
		{"open below", action.AddNewlineBelow},
		{"paste after", action.PasteAfter},
		{"paste before", action.PasteBefore},
		{"reindent", action.ReindentSelections},
		{"replace char", func(e *view.Editor) { action.ReplaceChar(e, 'x') }},
		{"replace yanked", action.ReplaceWithYanked},
		{"rotate contents backward", action.RotateContentsBackward},
		{"rotate contents forward", action.RotateContentsForward},
		{"smart tab", action.SmartTab},
		{"switch case", action.SwitchCase},
		{"lowercase", action.SwitchToLowercase},
		{"uppercase", action.SwitchToUppercase},
		{"toggle block comments", action.ToggleBlockComments},
		{"toggle comments", action.ToggleComments},
		{"toggle line comments", action.ToggleLineComments},
		{"unindent", action.Unindent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := editorWithReadOnlyText(t, "1\nabc\n")

			tc.fn(e)

			doc, ok := e.FocusedDocument()
			assert.True(t, ok)
			assert.Equal(t, "1\nabc\n", doc.Text().String())
		})
	}
}

func TestReadOnlyShellActions(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*view.Editor) error
	}{
		{"pipe", func(e *view.Editor) error {
			return action.ShellPipe(e, "cat")
		}},
		{"insert output", func(e *view.Editor) error {
			return action.ShellInsertOutput(e, "printf x")
		}},
		{"append output", func(e *view.Editor) error {
			return action.ShellAppendOutput(e, "printf x")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := editorWithReadOnlyText(t, "abc")

			err := tc.fn(e)

			assert.ErrorIs(t, err, view.ErrReadOnly)
		})
	}
}

func editorWithReadOnlyText(t *testing.T, text string) *view.Editor {
	t.Helper()
	e := editorWithText(t, text)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	setReadOnly(doc)
	return e
}

func setReadOnly(doc *view.Document) {
	f := reflect.ValueOf(doc).Elem().FieldByName("readonly")
	reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).
		Elem().
		SetBool(true)
}
