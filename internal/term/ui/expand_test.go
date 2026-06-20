package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
}
