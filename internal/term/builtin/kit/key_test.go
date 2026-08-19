package kit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

func TestRegisterHints(t *testing.T) {
	t.Run("empty store offers nothing", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		e.SetClipboard(testutil.NewFakeClipboard())

		assert.Empty(t, kit.RegisterHints(e))
	})

	t.Run("mirrored primary is offered once", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		e.WriteRegister(view.RegisterClipboard, []string{"yanked"})
		e.WriteRegister(view.RegisterPrimaryClipboard, []string{"yanked"})

		assert.Equal(t, []command.KeyHint{
			{Key: "+", Label: "yanked"},
		}, kit.RegisterHints(e))
	})

	t.Run("distinct primary is offered", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		e.WriteRegister(view.RegisterClipboard, []string{"copied"})
		e.WriteRegister(view.RegisterPrimaryClipboard, []string{"selected"})

		assert.Equal(t, []command.KeyHint{
			{Key: "*", Label: "selected"},
			{Key: "+", Label: "copied"},
		}, kit.RegisterHints(e))
	})

	t.Run("reads clipboard changed outside", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		e.WriteRegister(view.RegisterClipboard, []string{"stale"})
		clip.System = "fresh"

		assert.Equal(t, []command.KeyHint{
			{Key: "+", Label: "fresh"},
		}, kit.RegisterHints(e))
	})

	t.Run("named registers keep equal values", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		e.SetClipboard(testutil.NewFakeClipboard())
		e.WriteRegister('a', []string{"same"})
		e.WriteRegister('b', []string{"same"})

		assert.Equal(t, []command.KeyHint{
			{Key: "a", Label: "same"},
			{Key: "b", Label: "same"},
		}, kit.RegisterHints(e))
	})

	t.Run("preview collapses whitespace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		e.SetClipboard(testutil.NewFakeClipboard())
		e.WriteRegister('a', []string{"one\n\ttwo   three"})

		assert.Equal(t, []command.KeyHint{
			{Key: "a", Label: "one two three"},
		}, kit.RegisterHints(e))
	})
}

func TestKeyModifiers(t *testing.T) {
	t.Run("shift adds to a special binding", func(t *testing.T) {
		assert.Equal(t, command.KeyBinding{{{
			Code: command.KeyCode{Special: command.Tab},
			Mods: command.ModShift,
		}}}, kit.Shift(kit.Tab))
	})

	t.Run("alt adds to a special binding", func(t *testing.T) {
		assert.Equal(t, command.KeyBinding{{{
			Code: command.KeyCode{Special: command.Backspace},
			Mods: command.ModAlt,
		}}}, kit.AltKey(kit.Bksp))
	})

	t.Run("modifiers stack and leave the source alone", func(t *testing.T) {
		base := kit.Tab
		both := kit.AltKey(kit.Shift(base))

		assert.Equal(t,
			command.ModShift|command.ModAlt, both[0][0].Mods,
		)
		assert.Equal(t, command.ModNone, base[0][0].Mods)
	})

	t.Run("every alternative of an or is modified", func(t *testing.T) {
		shifted := kit.Shift(kit.Or(kit.Tab, kit.Bksp))

		assert.Len(t, shifted, 2)
		for _, seq := range shifted {
			assert.Equal(t, command.ModShift, seq[0].Mods)
		}
	})
}
