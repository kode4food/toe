// Package test is the shared black-box test harness for the default command
// modules: it builds an editor with all defaults registered and runs commands
// resolved from the keymaps, so each module's tests can live beside the module
// while exercising the fully assembled registry
package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

// Env builds an editor with all default commands registered, seeded with text.
// Commands resolved from the keymaps operate on the returned editor
func Env(t *testing.T, text string) (*view.Editor, *command.Keymaps) {
	t.Helper()
	km := command.NewKeymaps()
	dir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	e := view.NewEditor(dir)
	e.ResizeTree(80, 24)
	_, _ = builtin.Register(ui.New(e, km), km)
	if text != "" {
		testutil.SetEditorText(t, e, text)
	}
	return e, km
}

// EnvWithRegistry builds the same env as Env but also returns the registry so
// tests can call ApplyTOML to exercise section Apply funcs
func EnvWithRegistry(t *testing.T, text string) (
	*view.Editor, *command.Keymaps, *command.Registry,
) {
	t.Helper()
	km := command.NewKeymaps()
	dir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	e := view.NewEditor(dir)
	e.ResizeTree(80, 24)
	reg, _ := builtin.Register(ui.New(e, km), km)
	if text != "" {
		testutil.SetEditorText(t, e, text)
	}
	return e, km, reg
}

// TwoBufferEnv opens two files so buffer commands have multiple views to act on
func TwoBufferEnv(t *testing.T) (*view.Editor, *command.Keymaps) {
	t.Helper()
	dir := t.TempDir()
	km := command.NewKeymaps()
	e := view.NewEditor(dir)
	e.ResizeTree(80, 24)
	_, _ = builtin.Register(ui.New(e, km), km)
	for _, name := range []string{"a.txt", "b.txt"} {
		p := filepath.Join(dir, name)
		assert.NoError(t, os.WriteFile(p, []byte("x\n"), 0o644))
		_, err := e.OpenFile(p)
		assert.NoError(t, err)
	}
	return e, km
}

// RunCmd resolves a command by name and runs it against the editor. The default
// command wrappers ignore the Args, so nil is passed
func RunCmd(
	t *testing.T, km *command.Keymaps, e *view.Editor, name string,
) command.Result {
	t.Helper()
	cmd, ok := km.ResolveCommand(name)
	assert.True(t, ok)
	if cmd.Run == nil {
		return command.Result{}
	}
	return cmd.Run(e, nil)
}

// RunCmdArgs resolves a command and runs it with positional args parsed from
// input against the command's own signature
func RunCmdArgs(
	t *testing.T, km *command.Keymaps, e *view.Editor, name, input string,
) command.Result {
	t.Helper()
	cmd, ok := km.ResolveCommand(name)
	assert.True(t, ok)
	args, err := command.ParseArgs(input, cmd.Signature, false, nil)
	assert.NoError(t, err)
	return cmd.Run(e, args)
}

// DocText returns the focused document's full text
func DocText(t *testing.T, e *view.Editor) string {
	t.Helper()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	return doc.Text().String()
}

// CursorLine returns the focused document's primary cursor line
func CursorLine(t *testing.T, e *view.Editor) int {
	t.Helper()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	line, err := doc.Text().CharToLine(testutil.CursorPos(t, e))
	assert.NoError(t, err)
	return line
}

// MustFocusedView returns the focused view, failing the test if there is none
func MustFocusedView(t *testing.T, e *view.Editor) *view.View {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	return v
}

// Char is a bare key event for a rune, for feeding continuations
func Char(ch rune) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Char: ch}}
}

// Special is a bare key event for a named special key
func Special(name string) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Special: name}}
}
