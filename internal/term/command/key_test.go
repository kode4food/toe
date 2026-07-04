package command_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

func char(ch rune) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Char: ch}}
}

func special(name string) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Special: name}}
}

func TestKeyEventString(t *testing.T) {
	t.Run("plain char", func(t *testing.T) {
		assert.Equal(t, "a", char('a').String())
	})

	t.Run("special key", func(t *testing.T) {
		assert.Equal(t, "enter", special("enter").String())
	})

	t.Run("ctrl modifier", func(t *testing.T) {
		k := char('w').WithMods(command.ModCtrl)
		assert.Equal(t, "C-w", k.String())
	})

	t.Run("alt modifier", func(t *testing.T) {
		k := char('x').WithMods(command.ModAlt)
		assert.Equal(t, "A-x", k.String())
	})

	t.Run("shifted uppercase char", func(t *testing.T) {
		k := char('F').WithMods(command.ModShift)
		assert.Equal(t, "F", k.String())
	})

	t.Run("shifted unicode uppercase char", func(t *testing.T) {
		k := char('Å').WithMods(command.ModShift)
		assert.Equal(t, "Å", k.String())
	})

	t.Run("shifted non-uppercase char", func(t *testing.T) {
		k := char('!').WithMods(command.ModShift)
		assert.Equal(t, "S-!", k.String())
	})

	t.Run("ctrl shifted uppercase char", func(t *testing.T) {
		k := char('F').WithMods(command.ModCtrl | command.ModShift)
		assert.Equal(t, "C-S-f", k.String())
	})

	t.Run("ctrl+alt", func(t *testing.T) {
		k := char('a').WithMods(command.ModCtrl | command.ModAlt)
		s := k.String()
		assert.Contains(t, s, "C")
		assert.Contains(t, s, "A")
	})
}

func TestCommandRegistry(t *testing.T) {
	km := command.NewKeymaps()
	sig := command.Signature{
		Positionals: command.Positionals{Min: 1},
	}
	registered := command.Command{
		Aliases:   []string{"open", "o", "edit"},
		Signature: sig,
		Run: func(
			*view.Editor, *command.Args,
		) command.Result {
			return command.Result{}
		},
	}

	_ = km.Register("open", registered)
	cmd, ok := km.ResolveCommand("edit")
	list := km.Commands()

	assert.True(t, ok)
	assert.Equal(t, registered.Aliases, cmd.Aliases)
	assert.Equal(t, sig, cmd.Signature)
	assert.Equal(t, 1, len(list))
	assert.Equal(t, registered.Aliases, list[0].Aliases)
	assert.Equal(t, registered.Signature, list[0].Signature)
	assert.NotNil(t, list[0].Run)
}

func TestSparseCommands(t *testing.T) {
	t.Run("typed only", func(t *testing.T) {
		km := command.NewKeymaps()
		_ = km.Register("write", command.Command{
			Aliases: []string{"write", "w"},
			Run: func(
				*view.Editor, *command.Args,
			) command.Result {
				return command.Result{Message: "written"}
			},
		})

		cmd, ok := km.ResolveCommand("w")
		action, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			char('w'),
		})

		assert.True(t, ok)
		assert.NotNil(t, cmd.Run)
		assert.Nil(t, action)
		assert.False(t, found)
		assert.False(t, prefix)
	})

	t.Run("key only", func(t *testing.T) {
		called := false
		km := command.NewKeymaps()
		_ = km.Register("move-left", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				called = true
				return command.Result{}
			},
			Modes: []string{"NOR"},
			Keys:  map[string][]command.KeyBinding{"*": {{{char('h')}}}},
		})

		cmd, ok := km.ResolveCommand("move-left")
		action, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			char('h'),
		})

		assert.False(t, ok)
		assert.Nil(t, cmd.Run)
		assert.True(t, found)
		assert.False(t, prefix)
		action(nil)
		assert.True(t, called)
	})

	t.Run("typed and keyed", func(t *testing.T) {
		called := false
		km := command.NewKeymaps()
		_ = km.Register("quit", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				called = true
				return command.Result{Message: "quit"}
			},
			Modes:   []string{"NOR"},
			Keys:    map[string][]command.KeyBinding{"*": {{{char('q')}}}},
			Aliases: []string{"quit", "q"},
		})

		cmd, ok := km.ResolveCommand("q")
		action, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			char('q'),
		})

		assert.True(t, ok)
		assert.NotNil(t, cmd.Run)
		assert.True(t, found)
		assert.False(t, prefix)
		action(nil)
		assert.True(t, called)
	})

	t.Run("keyed and aliased in one registration", func(t *testing.T) {
		km := command.NewKeymaps()
		_ = km.Register("quit", command.Command{
			Run: func(_ *view.Editor, args *command.Args) command.Result {
				if args == nil {
					return command.Result{Message: "nil-safe"}
				}
				return command.Result{Message: "typed"}
			},
			Modes:   []string{"NOR"},
			Keys:    map[string][]command.KeyBinding{"*": {{{char('q')}}}},
			Aliases: []string{"quit", "q"},
		})

		cmd, ok := km.ResolveCommand("q")
		action, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			char('q'),
		})

		assert.True(t, ok)
		assert.NotNil(t, cmd.Run)
		assert.True(t, found)
		assert.False(t, prefix)
		assert.Nil(t, action(nil))
	})
}

func TestModeIsolation(t *testing.T) {
	t.Run("NOR key does not bleed into INS", func(t *testing.T) {
		km := command.NewKeymaps()
		run := func(*view.Editor, *command.Args) command.Result {
			return command.Result{}
		}
		_ = km.Register("move_char_left", command.Command{
			Run:   run,
			Modes: []string{"NOR", "INS"},
			Keys: map[string][]command.KeyBinding{
				"*":   {{{char('h')}, {special("left")}}},
				"INS": {{{special("left")}}},
			},
		})

		_, found, _ := km.Lookup(
			"INS", []command.KeyEvent{char('h')},
		)
		assert.False(t, found)

		_, found, _ = km.Lookup(
			"INS", []command.KeyEvent{special("left")},
		)
		assert.True(t, found)

		_, found, _ = km.Lookup("NOR", []command.KeyEvent{char('h')})
		assert.True(t, found)
	})
}

func TestIsTypable(t *testing.T) {
	t.Run("plain char is typable", func(t *testing.T) {
		assert.True(t, char('a').IsTypable())
	})

	t.Run("shift char is typable", func(t *testing.T) {
		assert.True(t,
			char('A').WithMods(command.ModShift).IsTypable(),
		)
	})

	t.Run("ctrl char is not typable", func(t *testing.T) {
		assert.False(t,
			char('c').WithMods(command.ModCtrl).IsTypable(),
		)
	})

	t.Run("alt char is not typable", func(t *testing.T) {
		assert.False(t,
			char('x').WithMods(command.ModAlt).IsTypable(),
		)
	})

	t.Run("special key is not typable", func(t *testing.T) {
		assert.False(t, special("enter").IsTypable())
	})
}

func TestKeyBind(t *testing.T) {
	called := false
	km := command.NewKeymaps()
	_ = km.Register("act", command.Command{
		Run: func(_ *view.Editor, _ *command.Args) command.Result {
			called = true
			return command.Result{}
		},
		Modes:   []string{"NOR"},
		Keys:    map[string][]command.KeyBinding{"*": {{{char('a')}}}},
		Aliases: []string{"act"},
	})

	t.Run("Bind adds extra sequence", func(t *testing.T) {
		km.Bind("NOR", "act", []command.KeyEvent{char('b')})
		a, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			char('b'),
		})
		assert.True(t, found)
		assert.False(t, prefix)
		called = false
		a(nil)
		assert.True(t, called)
	})

	t.Run("Bind unknown command is no-op", func(t *testing.T) {
		km.Bind("NOR", "nonexistent",
			[]command.KeyEvent{char('z')},
		)
		_, found, _ := km.Lookup("NOR", []command.KeyEvent{
			char('z'),
		})
		assert.False(t, found)
	})

	t.Run("Bind command without Run is no-op", func(t *testing.T) {
		km2 := command.NewKeymaps()
		_ = km2.Register("norun", command.Command{
			Modes: []string{"NOR"},
			Keys:  map[string][]command.KeyBinding{"*": {{{char('x')}}}},
		})
		km2.Bind("NOR", "norun",
			[]command.KeyEvent{char('y')},
		)
		_, found, _ := km2.Lookup("NOR", []command.KeyEvent{
			char('y'),
		})
		assert.False(t, found)
	})
}

func TestLabelNode(t *testing.T) {
	km := command.NewKeymaps()
	_ = km.Register("goto-file", command.Command{
		Run: func(*view.Editor, *command.Args) command.Result {
			return command.Result{}
		},
		Modes: []string{"NOR"},
		Keys: map[string][]command.KeyBinding{
			"*": {{{char('g'), char('f')}}},
		},
	})

	t.Run("sets label on prefix node", func(t *testing.T) {
		km.LabelNode("NOR", []command.KeyEvent{char('g')}, "Goto")
		title, hints := km.PendingHints("NOR", []command.KeyEvent{
			char('g'),
		})
		assert.Equal(t, "Goto", title)
		assert.Equal(t, 1, len(hints))
	})

	t.Run("LabelNode on unknown mode is no-op", func(t *testing.T) {
		km.LabelNode("UNK", []command.KeyEvent{char('g')}, "X")
		title, hints := km.PendingHints("UNK", []command.KeyEvent{
			char('g'),
		})
		assert.Equal(t, "", title)
		assert.Nil(t, hints)
	})

	t.Run("LabelNode on nonexistent key is no-op", func(t *testing.T) {
		km.LabelNode("NOR", []command.KeyEvent{char('z')}, "Z")
		_, hints := km.PendingHints("NOR", []command.KeyEvent{
			char('z'),
		})
		assert.Nil(t, hints)
	})
}

func TestPendingHints(t *testing.T) {
	km := command.NewKeymaps()
	run := func(*view.Editor, *command.Args) command.Result {
		return command.Result{}
	}
	_ = km.Register("ga", command.Command{
		Run:   run,
		Modes: []string{"NOR"},
		Keys: map[string][]command.KeyBinding{
			"*": {{{char('g'), char('a')}}},
		},
	})
	_ = km.Register("gb", command.Command{
		Run:   run,
		Modes: []string{"NOR"},
		Keys: map[string][]command.KeyBinding{
			"*": {{{char('g'), char('b')}}},
		},
	})
	_ = km.Register("gF", command.Command{
		Run:   run,
		Modes: []string{"NOR"},
		Keys: map[string][]command.KeyBinding{
			"*": {{{char('g'), char('F').WithMods(command.ModShift)}}},
		},
	})

	t.Run("returns hints for prefix", func(t *testing.T) {
		_, hints := km.PendingHints("NOR", []command.KeyEvent{
			char('g'),
		})
		assert.Equal(t, 3, len(hints))
	})

	t.Run("displays shifted uppercase char", func(t *testing.T) {
		_, hints := km.PendingHints("NOR", []command.KeyEvent{
			char('g'),
		})
		assert.Contains(t, hints, command.KeyHint{Key: "F", Label: "F"})
	})

	t.Run("returns empty for unknown mode", func(t *testing.T) {
		title, hints := km.PendingHints("UNK", []command.KeyEvent{
			char('g'),
		})
		assert.Equal(t, "", title)
		assert.Nil(t, hints)
	})

	t.Run("returns empty at leaf node", func(t *testing.T) {
		_, hints := km.PendingHints("NOR", []command.KeyEvent{
			char('g'), char('a'),
		})
		assert.Nil(t, hints)
	})

	t.Run("returns empty for unknown key in mode", func(t *testing.T) {
		_, hints := km.PendingHints("NOR", []command.KeyEvent{
			char('z'),
		})
		assert.Nil(t, hints)
	})
}

func TestKeyModifiers(t *testing.T) {
	t.Run("Has returns true for set bit", func(t *testing.T) {
		m := command.ModCtrl | command.ModAlt
		assert.True(t, m.Has(command.ModCtrl))
		assert.True(t, m.Has(command.ModAlt))
		assert.False(t, m.Has(command.ModShift))
	})

	t.Run("HasOnly matches exact bits", func(t *testing.T) {
		assert.True(t, command.ModShift.HasOnly(command.ModShift))
		assert.True(t,
			(command.ModCtrl | command.ModShift).
				HasOnly(command.ModCtrl|command.ModShift),
		)
		assert.False(t,
			(command.ModCtrl | command.ModShift).HasOnly(command.ModShift),
		)
		assert.False(t, command.ModNone.HasOnly(command.ModShift))
	})
}

func TestKeymapsBindAndLookup(t *testing.T) {
	var called string
	cmdQuit := func(_ *view.Editor) command.Continuation {
		called = "quit"
		return nil
	}
	cmdSave := func(_ *view.Editor) command.Continuation {
		called = "save"
		return nil
	}
	cmdGoTo := func(_ *view.Editor) command.Continuation {
		called = "goto"
		return nil
	}
	run := func(a command.KeyAction) command.Run {
		return func(e *view.Editor, _ *command.Args) command.Result {
			return command.Result{Continuation: a(e)}
		}
	}

	km := command.NewKeymaps()
	_ = km.Register("quit", command.Command{
		Run:   run(cmdQuit),
		Modes: []string{"normal"},
		Keys:  map[string][]command.KeyBinding{"*": {{{char('q')}}}},
	})
	_ = km.Register("save", command.Command{
		Run:   run(cmdSave),
		Modes: []string{"normal"},
		Keys: map[string][]command.KeyBinding{
			"*": {{{char('w').WithMods(command.ModCtrl)}}},
		},
	})
	_ = km.Register("goto", command.Command{
		Run:   run(cmdGoTo),
		Modes: []string{"normal"},
		Keys: map[string][]command.KeyBinding{
			"*": {{{char('g'), char('g')}}},
		},
	})

	t.Run("single key binding found", func(t *testing.T) {
		a, found, prefix := km.Lookup("normal", []command.KeyEvent{char('q')})
		assert.True(t, found)
		assert.False(t, prefix)
		called = ""
		a(nil)
		assert.Equal(t, "quit", called)
	})

	t.Run("command name found", func(t *testing.T) {
		name, found, prefix := km.LookupCommand(
			"normal", []command.KeyEvent{char('q')},
		)
		assert.True(t, found)
		assert.False(t, prefix)
		assert.Equal(t, "quit", name)
	})

	t.Run("two-key sequence found", func(t *testing.T) {
		a, found, prefix := km.Lookup("normal", []command.KeyEvent{
			char('g'), char('g'),
		})
		assert.True(t, found)
		assert.False(t, prefix)
		called = ""
		a(nil)
		assert.Equal(t, "goto", called)
	})

	t.Run("prefix returns prefix=true", func(t *testing.T) {
		_, found, prefix := km.Lookup("normal", []command.KeyEvent{
			char('g'),
		})
		assert.False(t, found)
		assert.True(t, prefix)
	})

	t.Run("command prefix returns prefix=true", func(t *testing.T) {
		name, found, prefix := km.LookupCommand(
			"normal", []command.KeyEvent{char('g')},
		)
		assert.False(t, found)
		assert.True(t, prefix)
		assert.Empty(t, name)
	})

	t.Run("unknown key returns false", func(t *testing.T) {
		_, found, prefix := km.Lookup("normal", []command.KeyEvent{
			char('z'),
		})
		assert.False(t, found)
		assert.False(t, prefix)
	})

	t.Run("unknown mode returns false", func(t *testing.T) {
		_, found, prefix := km.Lookup("insert", []command.KeyEvent{char('q')})
		assert.False(t, found)
		assert.False(t, prefix)
	})
}
