package ale_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/ale"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

func TestRuntime(t *testing.T) {
	t.Run("binds command procedures", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		reg := command.NewRegistry(km)
		var got []string
		err := reg.RegisterCommand("record", command.Command{
			Modes:     command.DocModes,
			Signature: command.DefaultSignature(),
			Run: func(
				_ *view.Editor, args *command.Args,
			) command.Result {
				got = append(got, args.Positionals()...)
				return command.Result{}
			},
		})
		assert.NoError(t, err)
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		assert.NoError(t, execute(t, rt, `
			(toe/bind :mode :normal :key "C-x"
			  (toe/record "first")
			  (toe/record "second"))
		`))

		seq, err := command.ParseKeySequence("C-x")
		assert.NoError(t, err)
		lookup, ok := km.Lookup(view.ModeNormal, seq)
		assert.True(t, ok)
		assert.Nil(t, lookup.Action(e).Continuation)
		assert.Equal(t, []string{"first", "second"}, got)
	})

	t.Run("reports command errors", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		reg := command.NewRegistry(km)
		want := errors.New("failed")
		err := reg.RegisterCommand("fail", command.Command{
			Modes:     command.DocModes,
			Signature: command.DefaultSignature(),
			Run: func(
				*view.Editor, *command.Args,
			) command.Result {
				return command.Result{Error: want}
			},
		})
		assert.NoError(t, err)
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		assert.NoError(t, execute(t, rt,
			`(toe/bind :modes :normal :keys "x" (toe/fail))`,
		))

		seq, err := command.ParseKeySequence("x")
		assert.NoError(t, err)
		lookup, ok := km.Lookup(view.ModeNormal, seq)
		assert.True(t, ok)
		res := lookup.Action(e)
		assert.ErrorIs(t, res.Error, want)
	})

	t.Run("validates command calls", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		reg := command.NewRegistry(km)
		err := reg.RegisterCommand("record", command.Command{
			Modes: command.DocModes,
			Signature: command.Signature{
				Positionals: command.Positionals{Min: 1, Max: 1},
			},
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{}
			},
		})
		assert.NoError(t, err)
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		assert.Error(t, execute(t, rt, `(toe/record 1)`))
		assert.Error(t, execute(t, rt, `(toe/record "one" "two")`))
		assert.Error(t, execute(t, rt, `(toe/record)`))
		assert.NoError(t, execute(t, rt,
			`(eq (toe/record "one") (toe/record "one"))`,
		))
		e.SetMode(view.ModeTerminal)
		assert.Error(t, execute(t, rt, `(toe/record "one")`))
	})

	t.Run("binds modes and keys", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		reg := command.NewRegistry(km)
		err := reg.RegisterCommand("record", command.Command{
			Modes: command.DocModes,
			Run: func(
				*view.Editor, *command.Args,
			) command.Result {
				return command.Result{}
			},
		})
		assert.NoError(t, err)
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		assert.NoError(t, execute(t, rt, `
			(toe/bind :modes [:normal :insert :select "terminal" :image]
			          :keys ["g x" "g y"] :doc "Record"
			          (toe/record))
		`))

		for _, mode := range []view.Mode{
			view.ModeNormal, view.ModeInsert, view.ModeSelect,
			view.ModeTerminal, view.ModeImage,
		} {
			for _, key := range []string{"g x", "g y"} {
				seq, err := command.ParseKeySequence(key)
				assert.NoError(t, err)
				_, ok := km.Lookup(mode, seq)
				assert.True(t, ok)
			}
		}
		seq, err := command.ParseKeySequence("g")
		assert.NoError(t, err)
		_, hints := km.PendingHints(nil, view.ModeNormal, seq)
		assert.Contains(t, hints, command.KeyHint{
			Key: "x, y", Label: "Record",
		})
	})

	t.Run("preserves command signals", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		reg := command.NewRegistry(km)
		err := reg.RegisterCommand("quit", command.Command{
			Modes:     command.DocModes,
			Signature: command.DefaultSignature(),
			Run: func(
				*view.Editor, *command.Args,
			) command.Result {
				return command.Result{Signal: command.SignalQuit}
			},
		})
		assert.NoError(t, err)
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		assert.NoError(t, execute(t, rt,
			`(toe/bind :modes :normal :keys "q" (toe/quit))`,
		))

		seq, err := command.ParseKeySequence("q")
		assert.NoError(t, err)
		lookup, ok := km.Lookup(view.ModeNormal, seq)
		assert.True(t, ok)
		assert.Equal(t, command.SignalQuit, lookup.Action(e).Signal)
	})

	t.Run("rejects invalid binding", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		err = execute(t, rt,
			`(toe/bind :modes :unknown :keys "x" (lambda () 1))`,
		)
		assert.Error(t, err)
	})

	t.Run("rejects occupied binding", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		src := `(toe/bind :modes :normal :keys "x" 1)`
		assert.NoError(t, execute(t, rt, src))
		err = execute(t, rt, src)
		assert.Contains(t,
			i18n.ErrorText(err), i18n.Text(i18n.ErrorBindingExists),
		)
	})

	t.Run("returns action binding errors", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		assert.NoError(t, execute(t, rt, `
			(toe/bind :modes :normal :keys "x"
			  (toe/bind :modes :normal :keys "x" 1))
		`))
		seq, err := command.ParseKeySequence("x")
		assert.NoError(t, err)
		lookup, ok := km.Lookup(view.ModeNormal, seq)
		assert.True(t, ok)
		res := lookup.Action(e)
		assert.Contains(t,
			i18n.ErrorText(res.Error), i18n.Text(i18n.ErrorBindingExists),
		)
	})

	t.Run("ignores plain action values", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, km, _ := recordingRuntime(t, e)
		assert.NoError(t, execute(t, rt,
			`(toe/bind :modes :normal :keys "x" 1)`,
		))
		press(t, km, e, view.ModeNormal)
	})

	t.Run("rejects missing action body", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, err := ale.NewRuntime(e, command.NewKeymaps())
		assert.NoError(t, err)
		err = execute(t, rt,
			`(toe/bind :modes :normal :keys "x")`,
		)
		assert.Error(t, err)
	})

	t.Run("reports invalid arguments", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, err := ale.NewRuntime(e, command.NewKeymaps())
		assert.NoError(t, err)
		err = execute(t, rt,
			`(toe/bind :normal "x" (lambda () 1))`,
		)
		assert.Error(t, err)
		message := i18n.ErrorText(err)
		assert.Contains(t, message, "\narguments:")
		assert.Contains(t, message, `[:normal "x" `)
	})

	t.Run("rejects malformed options", func(t *testing.T) {
		tests := []struct {
			name string
			src  string
		}{
			{name: "unpaired", src: `(toe/bind* :modes :normal)`},
			{name: "invalid action", src: `(toe/bind*
				:modes :normal :keys "x" 1)`},
			{name: "option name", src: `(toe/bind* "modes" :normal :keys "x"
				(lambda (ctx) 1))`},
			{name: "duplicate", src: `(toe/bind :mode :normal :modes :insert
				:keys "x" 1)`},
			{name: "mode type", src: `(toe/bind :modes 1 :keys "x" 1)`},
			{name: "key type", src: `(toe/bind :modes :normal :keys 1 1)`},
			{name: "doc type", src: `(toe/bind
				:modes :normal :keys "x" :doc 1 1)`},
			{name: "when type", src: `(toe/bind*
				:modes :normal :keys "x" :when 1
				(lambda (ctx) 1))`},
			{name: "unknown", src: `(toe/bind
				:modes :normal :keys "x" :wat 1 1)`},
			{name: "invalid key", src: `(toe/bind
				:modes :normal :keys "C-no-such-key" 1)`},
			{name: "missing modes", src: `(toe/bind* :keys "x" :doc "x"
				(lambda (ctx) 1))`},
			{name: "empty modes", src: `(toe/bind :modes [] :keys "x" 1)`},
			{name: "missing keys", src: `(toe/bind* :modes :normal :doc "x"
				(lambda (ctx) 1))`},
			{name: "empty keys", src: `(toe/bind
				:modes :normal :keys [] 1)`},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				e := view.NewEditor(t.TempDir())
				rt, err := ale.NewRuntime(e, command.NewKeymaps())
				assert.NoError(t, err)
				assert.Error(t, execute(t, rt, test.src))
			})
		}
	})

	t.Run("isolates evaluations", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		rt, err := ale.NewRuntime(e, km)
		assert.NoError(t, err)
		assert.NoError(t, execute(t, rt, `(define leaked 42)`))
		assert.Error(t, execute(t, rt, `leaked`))
	})

}

func TestWhenBinding(t *testing.T) {
	t.Run("true predicate fires", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, km, got := recordingRuntime(t, e)
		assert.NoError(t, execute(t, rt, `
			(toe/bind :modes :normal :keys "x"
			  :when (eq :normal (:mode (:pane ctx)))
			  (toe/record "hit"))
		`))
		fireWhen(t, km, e)
		assert.Equal(t, []string{"hit"}, *got)
	})

	t.Run("false predicate blocks", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, km, got := recordingRuntime(t, e)
		assert.NoError(t, execute(t, rt, `
			(toe/bind :modes :normal :keys "x"
			  :when (eq :insert (:mode (:pane ctx)))
			  (toe/record "hit"))
		`))
		fireWhen(t, km, e)
		assert.Empty(t, *got)
	})

	t.Run("predicate error blocks", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		rt, km, _ := recordingRuntime(t, e)
		assert.NoError(t, execute(t, rt, `
			(toe/bind :modes :normal :keys "x"
			  :when (/ 1 0)
			  (toe/record "hit"))
		`))
		seq, err := command.ParseKeySequence("x")
		assert.NoError(t, err)
		lookup, ok := km.Lookup(view.ModeNormal, seq)
		assert.True(t, ok)
		assert.False(t, lookup.Enabled(e))
	})
}

// fireWhen dispatches "x" the way the UI does, honoring the :when gate
func fireWhen(t *testing.T, km *command.Keymaps, e *view.Editor) {
	t.Helper()
	seq, err := command.ParseKeySequence("x")
	assert.NoError(t, err)
	lookup, ok := km.Lookup(view.ModeNormal, seq)
	assert.True(t, ok)
	if lookup.Enabled(e) {
		lookup.Action(e)
	}
}

func execute(t *testing.T, rt *ale.Runtime, src string) error {
	t.Helper()
	return rt.Eval(src)
}
