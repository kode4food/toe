package defaults_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

// defaultsEnv builds an editor with all default commands registered, seeded
// with text. Commands resolved from the keymaps operate on the returned editor
func defaultsEnv(t *testing.T, text string) (*view.Editor, *command.Keymaps) {
	t.Helper()
	km := command.NewKeymaps()
	e := view.NewEditor(t.TempDir())
	_, _ = defaults.RegisterDefaults(ui.New(e, km), km)
	if text != "" {
		setText(t, e, text)
	}
	return e, km
}

func setText(t *testing.T, e *view.Editor, text string) {
	t.Helper()
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
}

func setCursor(t *testing.T, e *view.Editor, pos int) {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	doc.SetSelectionFor(v.ID(), core.PointSelection(pos))
}

func setSelection(
	t *testing.T, e *view.Editor, ranges []core.Range, primary int,
) {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	sel, err := core.NewSelection(ranges, primary)
	assert.NoError(t, err)
	doc.SetSelectionFor(v.ID(), sel)
}

func docText(t *testing.T, e *view.Editor) string {
	t.Helper()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	return doc.Text().String()
}

func cursorPos(t *testing.T, e *view.Editor) int {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	return doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
}

func cursorLine(t *testing.T, e *view.Editor) int {
	t.Helper()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	line, err := doc.Text().CharToLine(cursorPos(t, e))
	assert.NoError(t, err)
	return line
}

// runCmd resolves a command by name and runs it against the editor, returning
// its result. The default command wrappers ignore the Args, so nil is passed
func runCmd(
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

// twoBufferEnv opens two files so buffer commands have multiple views to act on
func twoBufferEnv(t *testing.T) (*view.Editor, *command.Keymaps) {
	t.Helper()
	dir := t.TempDir()
	km := command.NewKeymaps()
	e := view.NewEditor(dir)
	_, _ = defaults.RegisterDefaults(ui.New(e, km), km)
	for _, name := range []string{"a.txt", "b.txt"} {
		p := filepath.Join(dir, name)
		assert.NoError(t, os.WriteFile(p, []byte("x\n"), 0o644))
		_, err := e.OpenFile(p)
		assert.NoError(t, err)
	}
	return e, km
}

func mustFocusedView(t *testing.T, e *view.Editor) *view.View {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	return v
}

// envWithRegistry builds the same env as defaultsEnv but also returns the
// command.Registry so tests can call ApplyTOML to exercise section Apply fns
func envWithRegistry(t *testing.T, text string) (
	*view.Editor, *command.Keymaps, *command.Registry,
) {
	t.Helper()
	km := command.NewKeymaps()
	e := view.NewEditor(t.TempDir())
	reg, _ := defaults.RegisterDefaults(ui.New(e, km), km)
	if text != "" {
		setText(t, e, text)
	}
	return e, km, reg
}

func char(ch rune) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Char: ch}}
}

func special(name string) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Special: name}}
}

// runCmdArgs resolves a command and runs it with positional args parsed from
// input against the command's own signature
func runCmdArgs(
	t *testing.T, km *command.Keymaps, e *view.Editor, name, input string,
) command.Result {
	t.Helper()
	cmd, ok := km.ResolveCommand(name)
	assert.True(t, ok)
	args, err := command.ParseArgs(input, cmd.Signature, false, nil)
	assert.NoError(t, err)
	return cmd.Run(e, args)
}
