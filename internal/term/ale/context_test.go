package ale_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/ale"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

func TestContext(t *testing.T) {
	t.Run("reads cwd", func(t *testing.T) {
		dir := t.TempDir()
		e := view.NewEditor(dir)
		rt, km, got := recordingRuntime(t, e)
		bindContext(t, rt, `(toe/record (:cwd ctx))`)
		press(t, km, e, view.ModeNormal)
		assert.Equal(t, []string{dir}, *got)
	})

	t.Run("absent keys use the supplied default", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, km, got := recordingRuntime(t, e)
		bindContext(t, rt, `
			(toe/record (:missing ctx "fallback"))
			(toe/record (:path (:document (:pane ctx)) "none"))
		`)
		press(t, km, e, view.ModeNormal)
		assert.Equal(t, []string{"fallback", "none"}, *got)
	})

	t.Run("reports pane kind and mode", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, km, got := recordingRuntime(t, e)
		bindContext(t, rt, `
			(toe/record
			  (str (:kind (:pane ctx)))
			  (str (:mode (:pane ctx))))
		`)
		press(t, km, e, view.ModeNormal)
		assert.Equal(t, []string{":view", ":normal"}, *got)
	})

	t.Run("reflects mode changes", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, km, got := recordingRuntime(t, e)
		bindContext(t, rt, `(toe/record (str (:mode (:pane ctx))))`)
		press(t, km, e, view.ModeNormal)
		e.SetMode(view.ModeInsert)
		press(t, km, e, view.ModeInsert)
		assert.Equal(t, []string{":normal", ":insert"}, *got)
	})

	t.Run("reports document properties", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, km, got := recordingRuntime(t, e)
		bindContext(t, rt, `
			(let [doc (:document (:pane ctx))]
			  (toe/record
			    (:name doc)
			    (str (:modified doc))
			    (str (:read-only doc))))
		`)
		press(t, km, e, view.ModeNormal)
		testutil.SetEditorText(t, e, "edited")
		press(t, km, e, view.ModeNormal)
		assert.Equal(t, []string{
			"[scratch]", "#f", "#f",
			"[scratch]", "#t", "#f",
		}, *got)
	})

	t.Run("reports selection ranges", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		testutil.SetEditorText(t, e, "hello world")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 5),
			core.NewRange(6, 11),
		}, 1)
		rt, km, got := recordingRuntime(t, e)
		bindContext(t, rt, `
			(let [sel (:selection (:pane ctx))]
			  (toe/record
			    (str (:primary sel))
			    (str (length (:ranges sel)))
			    (str (:from (nth (:ranges sel) 0)))
			    (str (:to (nth (:ranges sel) 1)))
			    (str (:cursor (nth (:ranges sel) 1)))))
		`)
		press(t, km, e, view.ModeNormal)
		assert.Equal(t, []string{"1", "2", "0", "11", "10"}, *got)
	})
}

// bindContext binds the given body to "x" across the editing modes so a test
// can dispatch it and observe the injected ctx
func bindContext(t *testing.T, rt *ale.Runtime, body string) {
	t.Helper()
	assert.NoError(t, execute(t, rt, fmt.Sprintf(
		`(toe/bind :modes [:normal :insert :select] :keys "x" %s)`, body,
	)))
}

// press dispatches the "x" binding in the given mode, running its action
func press(t *testing.T, km *command.Keymaps, e *view.Editor, mode view.Mode) {
	t.Helper()
	seq, err := command.ParseKeySequence("x")
	assert.NoError(t, err)
	lookup, ok := km.Lookup(mode, seq)
	assert.True(t, ok)
	assert.NoError(t, lookup.Action(e).Error)
}

func recordingRuntime(
	t *testing.T, e *view.Editor,
) (*ale.Runtime, *command.Keymaps, *[]string) {
	t.Helper()
	km := command.NewKeymaps()
	reg := command.NewRegistry(km)
	got := &[]string{}
	err := reg.RegisterCommand("record", command.Command{
		Modes:     command.AllModes,
		Signature: command.DefaultSignature(),
		Run: func(_ *view.Editor, args *command.Args) command.Result {
			*got = append(*got, args.Positionals()...)
			return command.Result{}
		},
	})
	assert.NoError(t, err)
	rt, err := ale.NewRuntime(e, km)
	assert.NoError(t, err)
	return rt, km, got
}
