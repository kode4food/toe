package defaults_test

import (
	"slices"
	"strings"
	"testing"

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

	t.Run("registers module options", func(t *testing.T) {
		reg := defaultRegistry(t)
		for _, key := range []string{
			"editor.scrolloff",
			"editor.search.smart-case",
			"editor.auto-pairs",
			"editor.shell",
			"editor.cursorline",
			"editor.mouse",
			"theme",
		} {
			_, ok := reg.LookupOption(key)
			assert.True(t, ok)
		}
	})

	t.Run("binds window rotation", func(t *testing.T) {
		km := defaultKeymaps(t)
		for _, seq := range [][]command.KeyEvent{
			{ctrl('w'), command.Char('w')},
			{ctrl('w'), ctrl('w')},
			{command.Char(' '), command.Char('w'), command.Char('w')},
			{command.Char(' '), command.Char('w'), ctrl('w')},
		} {
			_, found, prefix := km.Lookup("NOR", seq)
			assert.True(t, found)
			assert.False(t, prefix)
		}
	})

	t.Run("buffer next distinct", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, ok := km.ResolveCommand("goto_next_buffer")
		assert.True(t, ok)
		_, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			command.Char('g'), command.Char('n'),
		})
		assert.True(t, found)
		assert.False(t, prefix)
	})

	t.Run("insert end newline-aware", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, ok := km.ResolveCommand("goto_line_end_newline")
		assert.True(t, ok)
		_, found, prefix := km.Lookup("INS", []command.KeyEvent{
			command.Special("end"),
		})
		assert.True(t, found)
		assert.False(t, prefix)
	})

	t.Run("paragraph keys use unimpaired prefixes", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			command.Char('['),
		})
		assert.False(t, found)
		assert.True(t, prefix)

		_, found, prefix = km.Lookup("NOR", []command.KeyEvent{
			command.Char('['), command.Char('p'),
		})
		assert.True(t, found)
		assert.False(t, prefix)

		_, found, prefix = km.Lookup("NOR", []command.KeyEvent{
			command.Char(']'),
		})
		assert.False(t, found)
		assert.True(t, prefix)

		_, found, prefix = km.Lookup("NOR", []command.KeyEvent{
			command.Char(']'), command.Char('p'),
		})
		assert.True(t, found)
		assert.False(t, prefix)

		_, found, prefix = km.Lookup("NOR", []command.KeyEvent{
			command.Char('p'),
		})
		assert.True(t, found)
		assert.False(t, prefix)
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
		comps := cmd.Signature.Completer.Complete(e, cmd.Signature, "editor.sc")
		texts := make([]string, len(comps))
		for i, c := range comps {
			texts[i] = c.Text
		}
		assert.Contains(t, texts, "editor.scrolloff")
	})

	t.Run("toggle completes only bool option keys", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		cmd, ok := km.ResolveCommand("toggle_option")
		assert.True(t, ok)
		allComps := cmd.Signature.Completer.Complete(
			e, cmd.Signature, "editor.",
		)
		for _, c := range allComps {
			assert.NotEqual(t, "editor.scrolloff", c.Text)
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
	return command.Char(ch).WithMods(command.ModCtrl)
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
	if len(part) == 1 {
		return command.Char(rune(part[0]))
	}
	if !strings.HasPrefix(part, "<") || !strings.HasSuffix(part, ">") {
		return command.Special(part)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(part, "<"), ">")
	if inner == "space" {
		return command.Char(' ')
	}
	if value, ok := strings.CutPrefix(inner, "C-"); ok {
		return command.Char(rune(value[0])).WithMods(command.ModCtrl)
	}
	if value, ok := strings.CutPrefix(inner, "A-"); ok {
		return command.Char(rune(value[0])).WithMods(command.ModAlt)
	}
	if value, ok := strings.CutPrefix(inner, "S-"); ok {
		return command.Special(value).WithMods(command.ModShift)
	}
	return command.Special(inner)
}

func containsString(items []string, target string) bool {
	return slices.Contains(items, target)
}
