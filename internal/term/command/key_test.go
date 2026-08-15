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

func special(s command.Special) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Special: s}}
}

func TestKeyEventString(t *testing.T) {
	t.Run("plain char", func(t *testing.T) {
		assert.Equal(t, "a", char('a').String())
	})

	t.Run("special key", func(t *testing.T) {
		assert.Equal(t, "ret", special(command.Enter).String())
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

func TestParseKeySequence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []command.KeyEvent
	}{
		{
			name:  "characters",
			input: "g d",
			want:  []command.KeyEvent{char('g'), char('d')},
		},
		{
			name:  "modifiers",
			input: "C-S-f",
			want: []command.KeyEvent{{
				Code: command.KeyCode{Char: 'F'},
				Mods: command.ModCtrl | command.ModShift,
			}},
		},
		{
			name:  "special keys",
			input: "spc A-left",
			want: []command.KeyEvent{
				char(' '),
				{
					Code: command.KeyCode{Special: command.Left},
					Mods: command.ModAlt,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := command.ParseKeySequence(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("invalid key", func(t *testing.T) {
		_, err := command.ParseKeySequence("C-no-such-key")
		assert.ErrorIs(t, err, command.ErrInvalidKey)
	})
}

func TestCommandRegistry(t *testing.T) {
	km := command.NewKeymaps()
	sig := command.Signature{
		Positionals: command.Positionals{Min: 1},
	}
	registered := command.Command{
		Aliases:   []string{"open", "o", "edit"},
		Modes:     view.ModeNormal,
		Signature: sig,
		Run: func(
			*view.Editor, *command.Args,
		) command.Result {
			return command.Result{}
		},
	}

	_ = km.Register("open", registered)
	cmd := km.ResolveCommand("edit")

	assert.NotNil(t, cmd)
	assert.Equal(t, registered.Aliases, cmd.Aliases)
	assert.Equal(t, sig, cmd.Signature)
	assert.NotNil(t, cmd.Run)
}

func TestSparseCommands(t *testing.T) {
	t.Run("typed only", func(t *testing.T) {
		km := command.NewKeymaps()
		_ = km.Register("write", command.Command{
			Aliases: []string{"write", "w"},
			Modes:   view.ModeNormal,
			Run: func(
				*view.Editor, *command.Args,
			) command.Result {
				return command.Result{Message: "written"}
			},
		})

		cmd := km.ResolveCommand("w")
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('w'),
		})

		assert.NotNil(t, cmd)
		assert.NotNil(t, cmd.Run)
		assert.Nil(t, lookup.Action)
		assert.False(t, ok)
		assert.False(t, lookup.Prefix)
	})

	t.Run("missing modes errors", func(t *testing.T) {
		km := command.NewKeymaps()
		err := km.Register("write", command.Command{
			Aliases: []string{"write", "w"},
			Run: func(
				*view.Editor, *command.Args,
			) command.Result {
				return command.Result{}
			},
		})

		assert.ErrorIs(t, err, command.ErrNoModes)
	})

	t.Run("key only", func(t *testing.T) {
		called := false
		km := command.NewKeymaps()
		_ = km.Register("move-left", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				called = true
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char('h')}},
			},
		})

		cmd := km.ResolveCommand("move-left")
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('h'),
		})

		assert.NotNil(t, cmd)
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
		lookup.Action(nil)
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
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char('q')}},
			},
			Aliases: []string{"quit", "q"},
		})

		cmd := km.ResolveCommand("q")
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('q'),
		})

		assert.NotNil(t, cmd)
		assert.NotNil(t, cmd.Run)
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
		lookup.Action(nil)
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
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char('q')}},
			},
			Aliases: []string{"quit", "q"},
		})

		cmd := km.ResolveCommand("q")
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('q'),
		})

		assert.NotNil(t, cmd)
		assert.NotNil(t, cmd.Run)
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
		assert.Nil(t, lookup.Action(nil).Continuation)
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
			Modes: view.ModeNormal | view.ModeInsert,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny:    {{char('h')}, {special(command.Left)}},
				view.ModeInsert: {{special(command.Left)}},
			},
		})

		_, ok := km.Lookup(
			view.ModeInsert, []command.KeyEvent{char('h')},
		)
		assert.False(t, ok)

		_, ok = km.Lookup(
			view.ModeInsert, []command.KeyEvent{special(command.Left)},
		)
		assert.True(t, ok)

		_, ok = km.Lookup(view.ModeNormal, []command.KeyEvent{char('h')})
		assert.True(t, ok)
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
		assert.False(t, special(command.Enter).IsTypable())
	})
}

func TestPopOnBackspace(t *testing.T) {
	t.Run("pops plain backspace", func(t *testing.T) {
		called := false
		run := command.PopOnBackspace(func(
			*view.Editor, command.KeyEvent,
		) (command.Continuation, command.Transition) {
			called = true
			return nil, command.ContinuationDone
		})

		next, got := run(nil, special(command.Backspace))

		assert.Nil(t, next)
		assert.Equal(t, command.ContinuationPop, got)
		assert.False(t, called)
	})

	t.Run("passes modified backspace", func(t *testing.T) {
		called := false
		run := command.PopOnBackspace(func(
			*view.Editor, command.KeyEvent,
		) (command.Continuation, command.Transition) {
			called = true
			return nil, command.ContinuationDone
		})

		_, got := run(
			nil, special(command.Backspace).WithMods(command.ModCtrl),
		)

		assert.Equal(t, command.ContinuationDone, got)
		assert.True(t, called)
	})
}

func TestReadChar(t *testing.T) {
	t.Run("pops on backspace", func(t *testing.T) {
		run := command.ReadChar(func(
			*view.Editor, rune,
		) command.Continuation {
			return nil
		})

		_, got := run(nil, special(command.Backspace))

		assert.Equal(t, command.ContinuationPop, got)
	})

	t.Run("waits for plain character", func(t *testing.T) {
		called := false
		run := command.ReadChar(func(
			*view.Editor, rune,
		) command.Continuation {
			called = true
			return nil
		})

		_, got := run(nil, char('x').WithMods(command.ModCtrl))

		assert.Equal(t, command.ContinuationStay, got)
		assert.False(t, called)
	})

	t.Run("finishes on plain character", func(t *testing.T) {
		var gotChar rune
		run := command.ReadChar(func(
			_ *view.Editor, ch rune,
		) command.Continuation {
			gotChar = ch
			return nil
		})

		next, got := run(nil, char('x'))

		assert.Nil(t, next)
		assert.Equal(t, command.ContinuationDone, got)
		assert.Equal(t, 'x', gotChar)
	})

	t.Run("pushes returned continuation", func(t *testing.T) {
		want := command.Continuation(func(
			*view.Editor, command.KeyEvent,
		) (command.Continuation, command.Transition) {
			return nil, command.ContinuationDone
		})
		run := command.ReadChar(func(
			*view.Editor, rune,
		) command.Continuation {
			return want
		})

		next, got := run(nil, char('x'))

		assert.NotNil(t, next)
		assert.Equal(t, command.ContinuationPush, got)
	})
}

func TestKeyBind(t *testing.T) {
	called := false
	km := command.NewKeymaps()
	_ = km.Register("act", command.Command{
		Run: func(*view.Editor, *command.Args) command.Result {
			called = true
			return command.Result{}
		},
		Modes:   view.ModeNormal,
		Keys:    map[view.Mode]command.KeyBinding{view.ModeAny: {{char('a')}}},
		Aliases: []string{"act"},
	})

	t.Run("Bind adds extra sequence", func(t *testing.T) {
		km.Bind(view.ModeNormal, "act", []command.KeyEvent{char('b')})
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('b'),
		})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
		called = false
		lookup.Action(nil)
		assert.True(t, called)
	})

	t.Run("Bind unknown command is no-op", func(t *testing.T) {
		km.Bind(view.ModeNormal, "nonexistent",
			[]command.KeyEvent{char('z')},
		)
		_, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('z'),
		})
		assert.False(t, ok)
	})

	t.Run("command without Run still binds", func(t *testing.T) {
		km2 := command.NewKeymaps()
		_ = km2.Register("norun", command.Command{
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char('x')}},
			},
		})
		km2.Bind(view.ModeNormal, "norun",
			[]command.KeyEvent{char('y')},
		)
		for _, k := range []command.KeyEvent{char('x'), char('y')} {
			lookup, ok := km2.Lookup(view.ModeNormal,
				[]command.KeyEvent{k},
			)
			assert.True(t, ok)
			assert.Equal(t, "norun", lookup.Name)
			assert.Equal(t, command.Result{}, lookup.Action(nil))
		}
	})

	t.Run("BindResultAction adds sequence", func(t *testing.T) {
		action := func(*view.Editor) command.Result {
			return command.Result{Message: "custom"}
		}
		err := km.BindResultAction(command.BindActionArgs{
			Modes:  []view.Mode{view.ModeNormal},
			Action: action,
			Label:  "Custom",
			Seqs:   [][]command.KeyEvent{{char('c')}},
		})
		assert.NoError(t, err)
		lookup, ok := km.Lookup(
			view.ModeNormal, []command.KeyEvent{char('c')},
		)
		assert.True(t, ok)
		assert.Equal(t, "custom", lookup.Action(nil).Message)
	})

	t.Run("BindResultAction rejects duplicate", func(t *testing.T) {
		action := func(*view.Editor) command.Result {
			return command.Result{}
		}
		err := km.BindResultAction(command.BindActionArgs{
			Modes:  []view.Mode{view.ModeNormal},
			Action: action,
			Label:  "Duplicate",
			Seqs:   [][]command.KeyEvent{{char('a')}},
		})
		assert.ErrorIs(t, err, command.ErrBindingExists)
	})
}

func TestConditionalBinding(t *testing.T) {
	action := func(*view.Editor) command.Result {
		return command.Result{}
	}

	t.Run("nil when stays enabled", func(t *testing.T) {
		km := command.NewKeymaps()
		err := km.BindResultAction(command.BindActionArgs{
			Modes:  []view.Mode{view.ModeNormal},
			Action: action,
			Seqs:   [][]command.KeyEvent{{char('x')}},
		})
		assert.NoError(t, err)
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{char('x')})
		assert.True(t, ok)
		assert.True(t, lookup.Enabled(nil))
	})

	t.Run("false when disables match", func(t *testing.T) {
		km := command.NewKeymaps()
		err := km.BindResultAction(command.BindActionArgs{
			Modes:  []view.Mode{view.ModeNormal},
			Action: action,
			When:   func(*view.Editor) bool { return false },
			Seqs:   [][]command.KeyEvent{{char('x')}},
		})
		assert.NoError(t, err)
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{char('x')})
		assert.True(t, ok)
		assert.False(t, lookup.Enabled(nil))
	})

	t.Run("hints omit unavailable", func(t *testing.T) {
		km := command.NewKeymaps()
		yes := func(*view.Editor) bool { return true }
		no := func(*view.Editor) bool { return false }
		assert.NoError(t, km.BindResultAction(command.BindActionArgs{
			Modes:  []view.Mode{view.ModeNormal},
			Action: action,
			When:   yes,
			Label:  "Shown",
			Seqs:   [][]command.KeyEvent{{char(' '), char('a')}},
		}))
		assert.NoError(t, km.BindResultAction(command.BindActionArgs{
			Modes:  []view.Mode{view.ModeNormal},
			Action: action,
			When:   no,
			Label:  "Hidden",
			Seqs:   [][]command.KeyEvent{{char(' '), char('b')}},
		}))
		_, hints := km.PendingHints(
			nil, view.ModeNormal, []command.KeyEvent{char(' ')}, false,
		)
		labels := make([]string, len(hints))
		for i, h := range hints {
			labels[i] = h.Label
		}
		assert.Contains(t, labels, "Shown")
		assert.NotContains(t, labels, "Hidden")
	})
}

func TestLabelNode(t *testing.T) {
	km := command.NewKeymaps()
	_ = km.Register("goto-file", command.Command{
		Run: func(*view.Editor, *command.Args) command.Result {
			return command.Result{}
		},
		DocString: "File",
		Modes:     view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('g'), char('f')}},
		},
	})

	t.Run("sets label on prefix node", func(t *testing.T) {
		km.LabelNode(view.ModeNormal, command.KeyBinding{{char('g')}}, "Goto")
		title, hints := km.PendingHints(
			nil, view.ModeNormal, []command.KeyEvent{char('g')}, false,
		)

		assert.Equal(t, "Goto", title)
		assert.Equal(t, 1, len(hints))
	})

	t.Run("LabelNode on unknown mode is no-op", func(t *testing.T) {
		km.LabelNode(view.ModeBinary, command.KeyBinding{{char('g')}}, "X")
		title, hints := km.PendingHints(
			nil, view.ModeBinary, []command.KeyEvent{char('g')}, false,
		)

		assert.Equal(t, "", title)
		assert.Nil(t, hints)
	})

	t.Run("LabelNode on nonexistent key is no-op", func(t *testing.T) {
		km.LabelNode(view.ModeNormal, command.KeyBinding{{char('z')}}, "Z")
		_, hints := km.PendingHints(nil, view.ModeNormal, []command.KeyEvent{
			char('z'),
		}, false)

		assert.Nil(t, hints)
	})
}

func TestPendingHints(t *testing.T) {
	km := command.NewKeymaps()
	run := func(*view.Editor, *command.Args) command.Result {
		return command.Result{}
	}
	_ = km.Register("ga", command.Command{
		Run:       run,
		DocString: "A",
		Modes:     view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('g'), char('a')}},
		},
	})
	_ = km.Register("gb", command.Command{
		Run:       run,
		DocString: "B",
		Modes:     view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('g'), char('b')}},
		},
	})
	_ = km.Register("gF", command.Command{
		Run:       run,
		DocString: "F",
		Modes:     view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('g'), char('F').WithMods(command.ModShift)}},
		},
	})
	_ = km.Register("gc", command.Command{
		Run:   run,
		Modes: view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('g'), char('c')}},
		},
	})

	t.Run("returns hints for prefix", func(t *testing.T) {
		_, hints := km.PendingHints(nil, view.ModeNormal, []command.KeyEvent{
			char('g'),
		}, false)

		assert.Equal(t, 3, len(hints))
	})

	t.Run("displays shifted uppercase char", func(t *testing.T) {
		_, hints := km.PendingHints(nil, view.ModeNormal, []command.KeyEvent{
			char('g'),
		}, false)

		assert.Contains(t, hints, command.KeyHint{Key: "F", Label: "F"})
	})

	t.Run("omits undocumented commands", func(t *testing.T) {
		_, hints := km.PendingHints(nil, view.ModeNormal, []command.KeyEvent{
			char('g'),
		}, false)

		assert.NotContains(t, hints, command.KeyHint{Key: "c"})
	})

	t.Run("returns empty for unknown mode", func(t *testing.T) {
		title, hints := km.PendingHints(
			nil, view.ModeBinary, []command.KeyEvent{char('g')}, false,
		)

		assert.Equal(t, "", title)
		assert.Nil(t, hints)
	})

	t.Run("returns no child hints at leaf", func(t *testing.T) {
		_, hints := km.PendingHints(nil, view.ModeNormal, []command.KeyEvent{
			char('g'), char('a'),
		}, false)

		assert.Nil(t, hints)
	})

	t.Run("returns empty for unknown key in mode", func(t *testing.T) {
		_, hints := km.PendingHints(nil, view.ModeNormal, []command.KeyEvent{
			char('z'),
		}, false)

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

}

func TestKeymapsBindAndLookup(t *testing.T) {
	var called string
	cmdQuit := func(*view.Editor) command.Continuation {
		called = "quit"
		return nil
	}
	cmdSave := func(*view.Editor) command.Continuation {
		called = "save"
		return nil
	}
	cmdGoTo := func(*view.Editor) command.Continuation {
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
		Modes: view.ModeNormal,
		Keys:  map[view.Mode]command.KeyBinding{view.ModeAny: {{char('q')}}},
	})
	_ = km.Register("save", command.Command{
		Run:   run(cmdSave),
		Modes: view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('w').WithMods(command.ModCtrl)}},
		},
	})
	_ = km.Register("goto", command.Command{
		Run:   run(cmdGoTo),
		Modes: view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('g'), char('g')}},
		},
	})

	t.Run("single key binding found", func(t *testing.T) {
		lookup, ok := km.Lookup(
			view.ModeNormal, []command.KeyEvent{char('q')},
		)
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
		called = ""
		lookup.Action(nil)
		assert.Equal(t, "quit", called)
	})

	t.Run("command name found", func(t *testing.T) {
		lookup, ok := km.Lookup(
			view.ModeNormal, []command.KeyEvent{char('q')},
		)
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
		assert.Equal(t, "quit", lookup.Name)
	})

	t.Run("two-key sequence found", func(t *testing.T) {
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('g'), char('g'),
		})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
		called = ""
		lookup.Action(nil)
		assert.Equal(t, "goto", called)
	})

	t.Run("prefix returns prefix=true", func(t *testing.T) {
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('g'),
		})
		assert.False(t, ok)
		assert.True(t, lookup.Prefix)
	})

	t.Run("command prefix returns prefix=true", func(t *testing.T) {
		lookup, ok := km.Lookup(
			view.ModeNormal, []command.KeyEvent{char('g')},
		)
		assert.False(t, ok)
		assert.True(t, lookup.Prefix)
		assert.Empty(t, lookup.Name)
	})

	t.Run("unknown key returns false", func(t *testing.T) {
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			char('z'),
		})
		assert.False(t, ok)
		assert.False(t, lookup.Prefix)
	})

	t.Run("unknown mode returns false", func(t *testing.T) {
		lookup, ok := km.Lookup(view.ModeInsert, []command.KeyEvent{char('q')})
		assert.False(t, ok)
		assert.False(t, lookup.Prefix)
	})
}

func TestKeymapsModeFilters(t *testing.T) {
	km := command.NewKeymaps()
	run := func(*view.Editor, *command.Args) command.Result {
		return command.Result{}
	}
	_ = km.Register("normal", command.Command{
		Run:     run,
		Modes:   view.ModeNormal,
		Aliases: []string{"n"},
	})
	_ = km.Register("image", command.Command{
		Run:     run,
		Modes:   view.ModeImage,
		Aliases: []string{"i"},
	})

	assert.NotNil(t, km.ResolveCommandIn(view.ModeNormal, "n"))
	assert.Nil(t, km.ResolveCommandIn(view.ModeImage, "n"))

	cmds := km.CommandsIn(view.ModeImage)
	assert.Len(t, cmds, 1)
	assert.Equal(t, "image", cmds[0].Name)
}
