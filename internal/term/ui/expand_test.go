package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestTokenExpander(t *testing.T) {
	t.Run("unquoted token is returned as-is", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:    command.TokenUnquoted,
			Content: "hello",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("quoted token is returned as-is", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:    command.TokenQuoted,
			Content: "hello world",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello world", result)
	})

	t.Run("unicode expansion decodes hex codepoint", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionUnicode,
			Content:   "2665",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "♥", result)
	})

	t.Run("rejects invalid unicode codepoint", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionUnicode,
			Content:   "xxxxxx",
		}
		_, err := expand(tok)
		assert.Error(t, err)
	})

	t.Run("register expansion reads register value", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Registers().Write('a', []string{"hello"})
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionRegister,
			Content:   "a",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("unset register expands empty", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionRegister,
			Content:   "z",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("variable expansion resolves cursor_line", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.NewDocument()
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionVariable,
			Content:   "cursor_line",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "1", result)
	})

	t.Run("current_working_directory variable", func(t *testing.T) {
		dir := t.TempDir()
		e := view.NewEditor(dir)
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionVariable,
			Content:   "current_working_directory",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, dir, result)
	})

	t.Run("rejects unknown variable", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.NewDocument()
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionVariable,
			Content:   "no_such_var",
		}
		_, err := expand(tok)
		assert.Error(t, err)
	})

	t.Run("double-quoted percent markers expand", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Registers().Write('a', []string{"world"})
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:    command.TokenExpand,
			Content: "hello %reg{a}",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello world", result)
	})

	t.Run("expand inner handles escaped percent", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:    command.TokenExpand,
			Content: "100%% done",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "100% done", result)
	})

	t.Run("runs command and trims newline", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionShell,
			Content:   "echo hello",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("default token kind returns content", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		result, err := expand(command.Token{
			Kind:    command.TokenKind(99),
			Content: "raw",
		})
		assert.NoError(t, err)
		assert.Equal(t, "raw", result)
	})
}

func TestExpandVariables(t *testing.T) {
	t.Run("cursor_column on line start", func(t *testing.T) {
		e := expandEditorWithText(t, "hello\nworld\n")
		result, err := expandVar(t, e, "cursor_column")
		assert.NoError(t, err)
		assert.Equal(t, "1", result)
	})

	t.Run("buffer_name scratch returns name", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.NewDocument()
		result, err := expandVar(t, e, "buffer_name")
		assert.NoError(t, err)
		assert.Equal(t, view.ScratchBufferName, result)
	})

	t.Run("file_path_absolute scratch returns cwd", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.NewDocument()
		result, err := expandVar(t, e, "file_path_absolute")
		assert.NoError(t, err)
		assert.Equal(t, e.Cwd(), result)
	})

	t.Run("line_ending returns lf by default", func(t *testing.T) {
		e := expandEditorWithText(t, "abc\n")
		result, err := expandVar(t, e, "line_ending")
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("workspace_directory returns a dir", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.NewDocument()
		result, err := expandVar(t, e, "workspace_directory")
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("language default is text", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.NewDocument()
		result, err := expandVar(t, e, "language")
		assert.NoError(t, err)
		assert.Equal(t, "text", result)
	})

	t.Run("selection returns selected text", func(t *testing.T) {
		e := expandEditorWithText(t, "hello world\n")
		doc, _ := e.FocusedDocument()
		v, _ := e.FocusedView()
		doc.SetSelectionFor(v.ID(), core.SingleSelection(0, 5))
		result, err := expandVar(t, e, "selection")
		assert.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("selection_line_start returns line", func(t *testing.T) {
		e := expandEditorWithText(t, "hello\n")
		result, err := expandVar(t, e, "selection_line_start")
		assert.NoError(t, err)
		assert.Equal(t, "1", result)
	})

	t.Run("selection_line_end returns line", func(t *testing.T) {
		e := expandEditorWithText(t, "hello\n")
		result, err := expandVar(t, e, "selection_line_end")
		assert.NoError(t, err)
		assert.Equal(t, "1", result)
	})
}

func expandVar(
	t *testing.T, e *view.Editor, name string,
) (string, error) {
	t.Helper()
	expand := ui.NewTokenExpander(e)
	return expand(command.Token{
		Kind:      command.TokenExpansion,
		Expansion: command.ExpansionVariable,
		Content:   name,
	})
}

func expandEditorWithText(t *testing.T, text string) *view.Editor {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	e.NewDocument()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	rope := doc.Text()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, 0, text),
	})
	assert.NoError(t, err)
	tx := core.NewTransaction(rope).
		WithChanges(cs).
		WithSelection(core.PointSelection(0))
	assert.NoError(t, e.Apply(tx))
	return e
}
