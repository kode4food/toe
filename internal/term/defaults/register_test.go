package defaults_test

import (
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

	t.Run("keeps buffer next distinct from win rotation", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, ok := km.ResolveCommand("goto_next_buffer")
		assert.True(t, ok)
		_, found, prefix := km.Lookup("NOR", []command.KeyEvent{
			command.Char('g'), command.Char('n'),
		})
		assert.True(t, found)
		assert.False(t, prefix)
	})

	t.Run("binds insert end to newline-aware command", func(t *testing.T) {
		km := defaultKeymaps(t)

		_, ok := km.ResolveCommand("goto_line_end_newline")
		assert.True(t, ok)
		_, found, prefix := km.Lookup("INS", []command.KeyEvent{
			command.Special("end"),
		})
		assert.True(t, found)
		assert.False(t, prefix)
	})
}

func defaultKeymaps(t *testing.T) *command.Keymaps {
	t.Helper()
	km := command.NewKeymaps()
	e := view.NewEditor(t.TempDir())
	defaults.RegisterDefaults(ui.New(e, km), km)
	return km
}

func ctrl(ch rune) command.KeyEvent {
	return command.Char(ch).WithMods(command.ModCtrl)
}
