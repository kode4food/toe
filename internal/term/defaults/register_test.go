package defaults_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
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
			_, ok := km.ResolveCommand(name)
			assert.True(t, ok)
		}
	})

	t.Run("documented commands resolve", func(t *testing.T) {
		km := defaultKeymaps(t)
		for _, name := range documentedCommandNames(t) {
			t.Run(name, func(t *testing.T) {
				_, ok := km.ResolveCommand(name)
				assert.True(t, ok)
			})
		}
	})

	t.Run("registers module options", func(t *testing.T) {
		reg := defaultRegistry(t)
		for _, key := range []string{
			"scrolloff",
			"search.smart-case",
			"auto-pairs",
			"shell",
			"cursorline",
			"mouse",
			"theme",
		} {
			_, ok := reg.LookupOption(key)
			assert.True(t, ok)
		}
	})

	t.Run("binds window rotation", func(t *testing.T) {
		km := defaultKeymaps(t)
		for _, seq := range [][]command.KeyEvent{
			{ctrl('w'), char('w')},
			{ctrl('w'), ctrl('w')},
			{char(' '), char('w'), char('w')},
			{char(' '), char('w'), ctrl('w')},
		} {
			_, found, prefix := km.Lookup("NOR", seq)
			assert.True(t, found)
			assert.False(t, prefix)
		}
	})

	t.Run("buffer next distinct", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, ok := km.ResolveCommand("buffer_next")
		assert.True(t, ok)
		_, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			char('g'), char('n'),
		})
		assert.True(t, found)
		assert.False(t, prefix)
	})

	t.Run("insert end newline-aware", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, ok := km.ResolveCommand("goto_line_end_newline")
		assert.True(t, ok)
		_, found, prefix := km.Lookup("INS", []command.KeyEvent{
			special("end"),
		})
		assert.True(t, found)
		assert.False(t, prefix)
	})

	t.Run("capital bindings use shift", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			char(' '), char('F').WithMods(command.ModShift),
		})
		assert.True(t, found)
		assert.False(t, prefix)
	})

	t.Run("capital hints omit shift", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, hints := km.PendingHints("NOR", []command.KeyEvent{char(' ')})

		assert.Contains(t, hints, command.KeyHint{
			Key:   "F",
			Label: "Open file picker at current working directory",
		})
	})

	t.Run("space hints are ordered", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, hints := km.PendingHints("NOR", []command.KeyEvent{char(' ')})
		keys := make([]string, 0, len(hints))
		for _, h := range hints {
			keys = append(keys, h.Key)
		}

		assert.Equal(t, []string{
			"y", "Y", "p", "P", "R", "w",
			"h", "a", "k", "r", "s", "S",
			"f", "F", "e", ".", "b", "j",
			"d", "D", "/", "?", "'", "c",
			"A-c", "C", "g",
		}, keys)
	})

	t.Run("capital prefixes use shift", func(t *testing.T) {
		km := defaultKeymaps(t)

		title, hints := km.PendingHints("NOR", []command.KeyEvent{
			char('Z').WithMods(command.ModShift),
		})

		assert.Equal(t, "View", title)
		assert.NotNil(t, hints)
	})

	t.Run("paragraph keys use unimpaired prefixes", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			char('['),
		})
		assert.False(t, found)
		assert.True(t, prefix)

		_, found, prefix = km.Lookup("NOR", []command.KeyEvent{
			char('['), char('p'),
		})
		assert.True(t, found)
		assert.False(t, prefix)

		_, found, prefix = km.Lookup("NOR", []command.KeyEvent{
			char(']'),
		})
		assert.False(t, found)
		assert.True(t, prefix)

		_, found, prefix = km.Lookup("NOR", []command.KeyEvent{
			char(']'), char('p'),
		})
		assert.True(t, found)
		assert.False(t, prefix)

		_, found, prefix = km.Lookup("NOR", []command.KeyEvent{
			char('p'),
		})
		assert.True(t, found)
		assert.False(t, prefix)
	})

	t.Run("default keybindings resolve", func(t *testing.T) {
		km := defaultKeymaps(t)
		for _, cmd := range km.Commands() {
			name := commandName(cmd)
			for _, mode := range commandModes(cmd) {
				for _, binding := range commandBindings(cmd, mode) {
					for _, seq := range binding {
						t.Run(mode+"/"+name+"/"+keySeqString(seq),
							func(t *testing.T) {
								_, found, prefix := km.Lookup(mode, seq)
								assert.True(t, found)
								assert.False(t, prefix)
							},
						)
					}
				}
			}
		}
	})

	t.Run("no conflicting default keybindings", func(t *testing.T) {
		km := defaultKeymaps(t)
		seqs := collectDefaultKeySeqs(km)
		for key, names := range seqs {
			if len(names) < 2 || allowedDuplicateKey(key, names) {
				continue
			}
			assert.Failf(t, "duplicate key binding", "%s: %v", key, names)
		}
		for key := range seqs {
			mode, seq := splitKeySeq(key)
			events := parseKeySeq(t, seq)
			for other := range seqs {
				otherMode, otherSeq := splitKeySeq(other)
				if mode != otherMode || seq == otherSeq {
					continue
				}
				if !strings.HasPrefix(otherSeq, seq+" ") {
					continue
				}
				_, found, prefix := km.Lookup(mode, events)
				assert.False(t, found)
				assert.True(t, prefix)
			}
		}
	})
}

func TestOptionCompleters(t *testing.T) {
	t.Run("get completes all option keys", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		cmd, ok := km.ResolveCommand("get_option")
		assert.True(t, ok)
		comps := cmd.Signature.Completer.Complete(e, cmd.Signature, "sc")
		texts := make([]string, len(comps))
		for i, c := range comps {
			texts[i] = c.Text
		}
		assert.Contains(t, texts, "scrolloff")
	})

	t.Run("toggle completes only bool option keys", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		cmd, ok := km.ResolveCommand("toggle_option")
		assert.True(t, ok)
		allComps := cmd.Signature.Completer.Complete(
			e, cmd.Signature, "",
		)
		for _, c := range allComps {
			assert.NotEqual(t, "scrolloff", c.Text)
		}
		assert.True(t, len(allComps) > 0)
	})
}

func defaultKeymaps(t *testing.T) *command.Keymaps {
	t.Helper()
	km := command.NewKeymaps()
	e := view.NewEditor(t.TempDir())
	_, _ = defaults.RegisterDefaults(ui.New(e, km), km)
	return km
}

func defaultRegistry(t *testing.T) *command.Registry {
	t.Helper()
	km := command.NewKeymaps()
	e := view.NewEditor(t.TempDir())
	reg, err := defaults.RegisterDefaults(ui.New(e, km), km)
	assert.NoError(t, err)
	return reg
}

func ctrl(ch rune) command.KeyEvent {
	return char(ch).WithMods(command.ModCtrl)
}

func collectDefaultKeySeqs(km *command.Keymaps) map[string][]string {
	seqs := map[string][]string{}
	for _, cmd := range km.Commands() {
		name := commandName(cmd)
		for _, mode := range commandModes(cmd) {
			for _, binding := range commandBindings(cmd, mode) {
				for _, seq := range binding {
					key := mode + "\t" + keySeqString(seq)
					seqs[key] = append(seqs[key], name)
				}
			}
		}
	}
	return seqs
}

func commandName(cmd command.Command) string {
	if len(cmd.Aliases) == 0 {
		return ""
	}
	return cmd.Aliases[0]
}

func commandModes(cmd command.Command) []string {
	if len(cmd.Modes) == 0 {
		return []string{"NOR", "SEL", "INS"}
	}
	return cmd.Modes
}

func commandBindings(cmd command.Command, mode string) []command.KeyBinding {
	if bindings, ok := cmd.Keys[mode]; ok {
		return bindings
	}
	return cmd.Keys["*"]
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

func allowedDuplicateKey(key string, names []string) bool {
	_, seq := splitKeySeq(key)
	if seq != "esc" || len(names) != 2 {
		return false
	}
	return containsString(names, "normal_mode") &&
		containsString(names, "exit_select_mode")
}

func splitKeySeq(key string) (string, string) {
	mode, seq, _ := strings.Cut(key, "\t")
	return mode, seq
}

func parseKeySeq(t *testing.T, seq string) []command.KeyEvent {
	t.Helper()
	if seq == "" {
		return nil
	}
	parts := strings.Split(seq, " ")
	out := make([]command.KeyEvent, 0, len(parts))
	for _, part := range parts {
		out = append(out, parseKeyPart(t, part))
	}
	return out
}

func parseKeyPart(t *testing.T, part string) command.KeyEvent {
	t.Helper()
	runes := []rune(part)
	if len(runes) == 1 {
		r := runes[0]
		if unicode.IsUpper(r) {
			return char(r).WithMods(command.ModShift)
		}
		return char(r)
	}
	if !strings.HasPrefix(part, "<") || !strings.HasSuffix(part, ">") {
		return special(part)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(part, "<"), ">")
	if inner == "space" {
		return char(' ')
	}
	if value, ok := strings.CutPrefix(inner, "C-"); ok {
		return char(rune(value[0])).WithMods(command.ModCtrl)
	}
	if value, ok := strings.CutPrefix(inner, "A-"); ok {
		return char(rune(value[0])).WithMods(command.ModAlt)
	}
	if value, ok := strings.CutPrefix(inner, "S-"); ok {
		r := []rune(value)
		if len(r) == 1 {
			return char(r[0]).WithMods(command.ModShift)
		}
		return special(value).WithMods(command.ModShift)
	}
	return special(inner)
}

func containsString(items []string, target string) bool {
	return slices.Contains(items, target)
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
