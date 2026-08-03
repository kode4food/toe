package builtin_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type (
	keySeqKey struct {
		mode view.Mode
		seq  string
	}

	keySeqInfo struct {
		names  []string
		events []command.KeyEvent
	}
)

func TestDefaults(t *testing.T) {
	t.Run("registers command-line actions", func(t *testing.T) {
		km := defaultKeymaps(t)
		for _, name := range []string{
			"move_prev_word_end",
			"move_prev_long_word_end",
			"move_next_sub_word_start",
			"move_prev_sub_word_start",
			"move_next_sub_word_end",
			"move_prev_sub_word_end",
			"extend_to_first_nonwhitespace",
			"extend_prev_word_end",
			"extend_prev_long_word_end",
			"extend_next_sub_word_start",
			"extend_prev_sub_word_start",
			"extend_next_sub_word_end",
			"extend_prev_sub_word_end",
			"extend_to_file_end",
			"make_search_word_bounded",
			"extend_to_line_end_newline",
			"half_page_up",
			"half_page_down",
			"select_line_above",
			"select_line_below",
			"reflow",
			"wclose!",
		} {
			assert.NotNil(t, km.ResolveCommand(name))
		}
	})

	t.Run("documented commands resolve", func(t *testing.T) {
		km := defaultKeymaps(t)
		for _, name := range documentedCommandNames(t) {
			t.Run(name, func(t *testing.T) {
				assert.NotContains(t, name, "_")
				assert.NotNil(t, km.ResolveCommand(name))
			})
		}
	})

	t.Run("every command is documented", func(t *testing.T) {
		km := defaultKeymaps(t)
		documented := map[string]bool{}
		for _, name := range documentedCommandNames(t) {
			documented[name] = true
		}
		for _, cmd := range allCommands(km) {
			name := commandName(cmd)
			t.Run(name, func(t *testing.T) {
				assert.True(t, documented[name])
			})
		}
	})

	t.Run("registers module options", func(t *testing.T) {
		reg := defaultRegistry(t)
		for _, key := range []string{
			"theme",
			"mouse",
			"middle-click-paste",
			"nerd-fonts",
			"insecure",
			"editor-config",
			"auto-session",
			"default-line-ending",
			"line-number",
			"cursorline",
			"cursorcolumn",
			"text-width",
			"rulers",
			"bufferline",
			"soft-wrap.enable",
			"soft-wrap.wrap-indicator",
			"soft-wrap.wrap-at-text-width",
			"whitespace.render",
			"whitespace.render.space",
			"whitespace.render.nbsp",
			"whitespace.render.tab",
			"whitespace.render.newline",
			"whitespace.characters.space",
			"whitespace.characters.nbsp",
			"whitespace.characters.tab",
			"whitespace.characters.tabpad",
			"whitespace.characters.newline",
			"indent-guides.render",
			"indent-guides.character",
			"indent-guides.skip-levels",
			"gutters.layout",
			"gutters.line-numbers.min-width",
			"auto-pairs",
			"continue-comments",
			"atomic-save",
			"insert-final-newline",
			"trim-final-newlines",
			"trim-trailing-whitespace",
			"auto-save.focus-lost",
			"auto-save.after-delay.enable",
			"auto-save.after-delay.timeout",
			"search.smart-case",
			"search.wrap-around",
			"scrolloff",
			"scroll-lines",
			"cursor-shape.normal",
			"cursor-shape.insert",
			"cursor-shape.select",
			"statusline.left",
			"statusline.right",
			"statusline.separator",
			"picker.split-ratios.diagnostics",
			"buffer-picker.start-position",
			"file-explorer.hidden",
			"file-explorer.follow-symlinks",
			"file-explorer.parents",
			"file-explorer.ignore-files",
			"file-explorer.flatten-dirs",
			"shell",
		} {
			t.Run(key, func(t *testing.T) {
				assert.NotNil(t, reg.LookupOption(key))
			})
		}
	})

	t.Run("binds window rotation", func(t *testing.T) {
		km := defaultKeymaps(t)
		for _, seq := range [][]command.KeyEvent{
			{ctrl('w'), test.Char('w')},
			{ctrl('w'), ctrl('w')},
			{test.Char(' '), test.Char('w'), test.Char('w')},
			{test.Char(' '), test.Char('w'), ctrl('w')},
		} {
			lookup, ok := km.Lookup(view.ModeNormal, seq)
			assert.True(t, ok)
			assert.False(t, lookup.Prefix)
		}
	})

	t.Run("buffer next distinct", func(t *testing.T) {
		km := defaultKeymaps(t)

		assert.NotNil(t, km.ResolveCommand("buffer_next"))
		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			test.Char('g'), test.Char('n'),
		})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
	})

	t.Run("insert end newline-aware", func(t *testing.T) {
		km := defaultKeymaps(t)

		assert.NotNil(t, km.ResolveCommand("goto_line_end_newline"))
		lookup, ok := km.Lookup(view.ModeInsert, []command.KeyEvent{
			test.Special(command.End),
		})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
	})

	t.Run("capital bindings use shift", func(t *testing.T) {
		km := defaultKeymaps(t)

		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			test.Char(' '), test.Char('F').WithMods(command.ModShift),
		})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
	})

	t.Run("capital hints omit shift", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, hints := km.PendingHints(nil,
			view.ModeNormal, []command.KeyEvent{test.Char(' ')},
		)

		assert.Contains(t, hints, command.KeyHint{
			Key:   "F",
			Label: "Open file picker at current working directory",
		})
	})

	t.Run("space hints are ordered", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, hints := km.PendingHints(nil,
			view.ModeNormal, []command.KeyEvent{test.Char(' ')},
		)
		keys := make([]string, 0, len(hints))
		for _, h := range hints {
			keys = append(keys, h.Key)
		}

		assert.Equal(t, []string{
			"w", "y", "Y", "p", "P", "R",
			"h", "a", "k", "r", "s", "S",
			"f", "F", "e", ".", "b", "j",
			"d", "D", "/", "?", "'", "c",
			"A-c", "C", "g",
		}, keys)
	})

	t.Run("leader aliases share menus", func(t *testing.T) {
		km := defaultKeymaps(t)

		for _, mode := range command.PaneModes.Split() {
			spaceTitle, spaceHints := km.PendingHints(nil,
				mode, []command.KeyEvent{test.Char(' ')},
			)
			aliasTitle, aliasHints := km.PendingHints(nil,
				mode, []command.KeyEvent{ctrl('\\')},
			)
			assert.Equal(t, spaceTitle, aliasTitle)
			assert.Equal(t, spaceHints, aliasHints)
		}
	})

	t.Run("terminal window menu mirrors other panes", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, hints := km.PendingHints(nil,
			view.ModeTerminal, []command.KeyEvent{ctrl('w')},
		)
		labels := make(map[string]string, len(hints))
		for _, h := range hints {
			labels[h.Key] = h.Label
		}

		// terminal-specific addition
		assert.Equal(t, "Search focused terminal's scrollback", labels["/"])
		// pane-management commands shared with every other pane
		assert.Equal(t, "Create a new scratch buffer", labels["n"])
		assert.Equal(t, "Vertical right split", labels["v, C-v"])
	})

	t.Run("terminal space menu is filtered", func(t *testing.T) {
		km := defaultKeymaps(t)

		title, hints := km.PendingHints(nil, view.ModeTerminal, []command.KeyEvent{
			test.Char(' '),
		})
		labels := make(map[string]string, len(hints))
		for _, h := range hints {
			labels[h.Key] = h.Label
		}

		assert.Equal(t, "Leader", title)
		assert.Equal(t, "Window", labels["w"])
		assert.Equal(t, "Open file picker", labels["f"])
		assert.Equal(t, "Paste clipboard into terminal", labels["p"])
		assert.NotContains(t, labels, "y")
	})

	t.Run("read-only command prompts are bound", func(t *testing.T) {
		km := defaultKeymaps(t)

		for _, mode := range []view.Mode{view.ModeImage, view.ModeBinary} {
			lookup, ok := km.Lookup(mode, []command.KeyEvent{
				test.Char(':'),
			})

			assert.True(t, ok)
			assert.False(t, lookup.Prefix)
		}
	})

	t.Run("read-only pane commands are available", func(t *testing.T) {
		km := defaultKeymaps(t)
		names := []string{
			"open",
			"write-all",
			"write-all!",
			"write-quit-all",
			"write-quit-all!",
			"reload-all",
			"workspace-symbol-picker",
			"changed-file-picker",
			"change-directory",
			"save-session",
			"clear-register",
		}
		for _, mode := range []view.Mode{view.ModeImage, view.ModeBinary} {
			for _, name := range names {
				assert.NotNil(t, km.ResolveCommandIn(mode, name))
			}
		}
	})

	t.Run("image window hints are filtered", func(t *testing.T) {
		km := defaultKeymaps(t)

		title, hints := km.PendingHints(nil,
			view.ModeImage, []command.KeyEvent{ctrl('w')},
		)

		assert.Equal(t, "Window", title)
		assert.Contains(t, hints, command.KeyHint{
			Key:   "v, C-v",
			Label: "Vertical right split",
		})
		assert.Contains(t, hints, command.KeyHint{
			Key:   "q, C-q",
			Label: "Close window",
		})
		assert.NotContains(t, hints, command.KeyHint{
			Key:   "/",
			Label: "Search focused terminal's scrollback",
		})
	})

	t.Run("image space hints are filtered", func(t *testing.T) {
		km := defaultKeymaps(t)

		title, hints := km.PendingHints(nil, view.ModeImage, []command.KeyEvent{
			test.Char(' '),
		})

		assert.Equal(t, "Leader", title)
		assert.Contains(t, hints, command.KeyHint{
			Key:   "?",
			Label: "Open command palette",
		})
		assert.Contains(t, hints, command.KeyHint{
			Key:   "w",
			Label: "Window",
		})
		assert.NotContains(t, hints, command.KeyHint{
			Key:   "y",
			Label: "Yank selections to the clipboard",
		})
	})

	t.Run("capital prefixes use shift", func(t *testing.T) {
		km := defaultKeymaps(t)

		title, hints := km.PendingHints(nil, view.ModeNormal, []command.KeyEvent{
			test.Char('Z').WithMods(command.ModShift),
		})

		assert.Equal(t, "View", title)
		assert.NotNil(t, hints)
	})

	t.Run("paragraph keys use unimpaired prefixes", func(t *testing.T) {
		km := defaultKeymaps(t)

		lookup, ok := km.Lookup(view.ModeNormal, []command.KeyEvent{
			test.Char('['),
		})
		assert.False(t, ok)
		assert.True(t, lookup.Prefix)

		lookup, ok = km.Lookup(view.ModeNormal, []command.KeyEvent{
			test.Char('['), test.Char('p'),
		})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)

		lookup, ok = km.Lookup(view.ModeNormal, []command.KeyEvent{
			test.Char(']'),
		})
		assert.False(t, ok)
		assert.True(t, lookup.Prefix)

		lookup, ok = km.Lookup(view.ModeNormal, []command.KeyEvent{
			test.Char(']'), test.Char('p'),
		})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)

		lookup, ok = km.Lookup(view.ModeNormal, []command.KeyEvent{
			test.Char('p'),
		})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
	})

	t.Run("default keybindings resolve", func(t *testing.T) {
		km := defaultKeymaps(t)
		for _, cmd := range allCommands(km) {
			name := commandName(cmd)
			for _, mode := range commandModes(cmd) {
				for _, seq := range commandBindings(cmd, mode) {
					label := mode.String() + "/" + name +
						"/" + keySeqString(seq)
					t.Run(label, func(t *testing.T) {
						lookup, ok := km.Lookup(mode, seq)
						assert.True(t, ok)
						assert.False(t, lookup.Prefix)
					})
				}
			}
		}
	})

	t.Run("no conflicting default keybindings", func(t *testing.T) {
		km := defaultKeymaps(t)
		seqs := collectDefaultKeySeqs(km)
		for key, info := range seqs {
			if len(info.names) < 2 || allowedDuplicateKey(key, info.names) {
				continue
			}
			assert.Failf(t, "duplicate key binding",
				"%s %s: %v", key.mode, key.seq, info.names,
			)
		}
		for key, info := range seqs {
			for other := range seqs {
				if key.mode != other.mode || key.seq == other.seq {
					continue
				}
				if !strings.HasPrefix(other.seq, key.seq+" ") {
					continue
				}
				lookup, ok := km.Lookup(key.mode, info.events)
				assert.False(t, ok)
				assert.True(t, lookup.Prefix)
			}
		}
	})
}

func TestOptionCompleters(t *testing.T) {
	t.Run("get completes all option keys", func(t *testing.T) {
		e, km := test.Env(t, "")
		cmd := km.ResolveCommand("get_option")
		assert.NotNil(t, cmd)
		comps := cmd.Signature.Completer.Complete(e, cmd.Signature, "sc")
		texts := make([]string, len(comps))
		for i, c := range comps {
			texts[i] = c.Text
		}
		assert.Contains(t, texts, "scrolloff")
	})

	t.Run("toggle completes only bool option keys", func(t *testing.T) {
		e, km := test.Env(t, "")
		cmd := km.ResolveCommand("toggle_option")
		assert.NotNil(t, cmd)
		allComps := cmd.Signature.Completer.Complete(
			e, cmd.Signature, "",
		)
		for _, c := range allComps {
			assert.NotEqual(t, "scrolloff", c.Text)
		}
		assert.True(t, len(allComps) > 0)
	})

	t.Run("set completes option keys", func(t *testing.T) {
		e, km := test.Env(t, "")
		cmd := km.ResolveCommand("set_option")
		assert.NotNil(t, cmd)
		comps := cmd.Signature.Completer.Complete(e, cmd.Signature, "sc")
		texts := make([]string, len(comps))
		for i, c := range comps {
			texts[i] = c.Text
		}
		assert.Contains(t, texts, "scrolloff")
	})

	t.Run("set completes option values", func(t *testing.T) {
		e, km := test.Env(t, "")
		cmd := km.ResolveCommand("set_option")
		assert.NotNil(t, cmd)
		cases := []struct{ input, want string }{
			{input: "theme moc", want: "mocha"},
			{input: "line-number r", want: "relative"},
			{input: "bufferline m", want: "multiple"},
			{input: "whitespace.render n", want: "none"},
			{input: "statusline.left [", want: `["mode"]`},
			{input: "gutters.layout [", want: `["diagnostics"]`},
			{input: "auto-pairs f", want: "false"},
			{input: "rulers ", want: "[]"},
		}
		for _, tc := range cases {
			t.Run(tc.input, func(t *testing.T) {
				comps := cmd.Signature.Completer.Complete(
					e, cmd.Signature, tc.input,
				)
				texts := make([]string, len(comps))
				for i, c := range comps {
					texts[i] = c.Text
				}
				assert.Contains(t, texts, tc.want)
			})
		}
	})
}

func defaultKeymaps(t *testing.T) *command.Keymaps {
	t.Helper()
	km := command.NewKeymaps()
	e := view.NewEditor(t.TempDir())
	_, _ = builtin.Register(ui.New(e, km), km)
	return km
}

func defaultRegistry(t *testing.T) *command.Registry {
	t.Helper()
	km := command.NewKeymaps()
	e := view.NewEditor(t.TempDir())
	reg, err := builtin.Register(ui.New(e, km), km)
	assert.NoError(t, err)
	return reg
}

func ctrl(ch rune) command.KeyEvent {
	return test.Char(ch).WithMods(command.ModCtrl)
}

func collectDefaultKeySeqs(km *command.Keymaps) map[keySeqKey]keySeqInfo {
	seqs := map[keySeqKey]keySeqInfo{}
	for _, cmd := range allCommands(km) {
		name := commandName(cmd)
		for _, mode := range commandModes(cmd) {
			for _, seq := range commandBindings(cmd, mode) {
				key := keySeqKey{mode: mode, seq: keySeqString(seq)}
				info := seqs[key]
				info.names = append(info.names, name)
				info.events = seq
				seqs[key] = info
			}
		}
	}
	return seqs
}

func commandName(cmd *command.Command) string {
	if len(cmd.Aliases) == 0 {
		return ""
	}
	return cmd.Aliases[0]
}

func commandModes(cmd *command.Command) []view.Mode {
	modes := cmd.Modes
	if modes == 0 {
		modes = command.DocModes
	}
	return modes.Split()
}

func allCommands(km *command.Keymaps) []*command.Command {
	return km.CommandsIn(command.AllModes | view.ModeCompletion)
}

func commandBindings(
	cmd *command.Command, mode view.Mode,
) command.KeyBinding {
	if bindings, ok := cmd.Keys[mode]; ok {
		return bindings
	}
	return cmd.Keys[view.ModeAny]
}

func keySeqString(seq []command.KeyEvent) string {
	parts := make([]string, 0, len(seq))
	for _, ev := range seq {
		if ev.Code.Char == ' ' && ev.Mods == command.ModNone {
			parts = append(parts, "<space>")
			continue
		}
		parts = append(parts, ev.String())
	}
	return strings.Join(parts, " ")
}

func allowedDuplicateKey(key keySeqKey, names []string) bool {
	if key.seq != "esc" || len(names) != 2 {
		return false
	}
	return slices.Contains(names, "normal-mode") &&
		slices.Contains(names, "exit-select-mode")
}

func documentedCommandNames(t *testing.T) []string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	assert.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	data, err := os.ReadFile(
		filepath.Join(root, "docs/content/docs/commands.md"),
	)
	assert.NoError(t, err)
	seen := map[string]bool{}
	var out []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 4 {
			continue
		}
		out = appendDocumentedCommandNames(out, seen, cells[1])
		out = appendDocumentedCommandNames(out, seen, cells[2])
	}
	return out
}

func appendDocumentedCommandNames(
	out []string, seen map[string]bool, cell string,
) []string {
	for {
		_, rest, ok := strings.Cut(cell, "`")
		if !ok {
			return out
		}
		name, after, ok := strings.Cut(rest, "`")
		if !ok {
			return out
		}
		cell = after
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
}
