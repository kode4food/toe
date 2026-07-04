package ui_test

import (
	"os"
	"path/filepath"
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

	t.Run("register expansion joins values", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Registers().Write('a', []string{"hello", "world"})
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionRegister,
			Content:   "a",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello\nworld", result)
	})

	t.Run("register expansion rejects names", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionRegister,
			Content:   "ab",
		}
		_, err := expand(tok)
		assert.Error(t, err)
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

	t.Run("variable with no document returns error", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionVariable,
			Content:   "cursor_line",
		}
		_, err := expand(tok)
		assert.ErrorIs(t, err, ui.ErrNoFocusedDocument)
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

	t.Run("shell expansion uses default shell", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/sh")
		e := view.NewEditor(t.TempDir())
		e.Options().Shell = nil
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionShell,
			Content:   "printf hello",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("shell expansion keeps stdout on error", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionShell,
			Content:   "printf hello; exit 1",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("shell expansion returns empty error", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionShell,
			Content:   "exit 1",
		}
		_, err := expand(tok)
		assert.Error(t, err)
	})

	t.Run("shell expansion expands inner tokens", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Registers().Write('a', []string{"hello"})
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionShell,
			Content:   "printf %reg{a}",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("shell expansion returns inner error", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionShell,
			Content:   "printf %var{missing}",
		}
		_, err := expand(tok)
		assert.Error(t, err)
	})

	t.Run("unknown expansion returns content", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:      command.TokenExpansion,
			Expansion: command.ExpansionKind(99),
			Content:   "raw",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "raw", result)
	})

	t.Run("trailing percent is preserved", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		expand := ui.NewTokenExpander(e)
		tok := command.Token{
			Kind:    command.TokenExpand,
			Content: "hello %",
		}
		result, err := expand(tok)
		assert.NoError(t, err)
		assert.Equal(t, "hello %", result)
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

	t.Run("buffer_name returns relative path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "file.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		result, err := expandVar(t, e, "buffer_name")
		assert.NoError(t, err)
		assert.Equal(t, "file.txt", result)
	})

	t.Run("file_path_absolute scratch returns cwd", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.NewDocument()
		result, err := expandVar(t, e, "file_path_absolute")
		assert.NoError(t, err)
		assert.Equal(t, e.Cwd(), result)
	})

	t.Run("file_path_absolute returns file path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "file.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		result, err := expandVar(t, e, "file_path_absolute")
		assert.NoError(t, err)
		assert.Equal(t, path, result)
	})

	t.Run("line_ending returns lf by default", func(t *testing.T) {
		e := expandEditorWithText(t, "abc\n")
		result, err := expandVar(t, e, "line_ending")
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("workspace_directory finds marker", func(t *testing.T) {
		root := t.TempDir()
		child := filepath.Join(root, "a", "b")
		assert.NoError(t, os.MkdirAll(child, 0o755))
		assert.NoError(t, os.Mkdir(filepath.Join(root, ".toe"), 0o755))
		e := view.NewEditor(child)
		e.NewDocument()
		result, err := expandVar(t, e, "workspace_directory")
		assert.NoError(t, err)
		assert.Equal(t, root, result)
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

	t.Run("language returns document language", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetLang("go")
		result, err := expandVar(t, e, "language")
		assert.NoError(t, err)
		assert.Equal(t, "go", result)
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

func expandVar(t *testing.T, e *view.Editor, name string) (string, error) {
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
