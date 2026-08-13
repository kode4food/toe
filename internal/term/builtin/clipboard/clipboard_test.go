package clipboard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

func TestClipboardYankPaste(t *testing.T) {
	t.Run("yank then paste after inserts text", func(t *testing.T) {
		e, km := test.Env(t, "abc\n")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)
		test.RunCmd(t, km, e, "yank")
		testutil.SetCursor(t, e, 3)
		test.RunCmd(t, km, e, "paste_after")
		assert.Contains(t, test.DocText(t, e), "abc")
	})

	t.Run("paste before inserts before cursor", func(t *testing.T) {
		e, km := test.Env(t, "abc\n")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)
		test.RunCmd(t, km, e, "yank")
		testutil.SetCursor(t, e, 4)
		test.RunCmd(t, km, e, "paste_before")
		assert.Contains(t, test.DocText(t, e), "abc")
	})

	t.Run("replace with yanked replaces selection", func(t *testing.T) {
		e, km := test.Env(t, "abc\ndef\n")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)
		test.RunCmd(t, km, e, "yank")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 4,
			Head:   7,
		}}, 0)
		test.RunCmd(t, km, e, "replace_with_yanked")
		assert.Contains(t, test.DocText(t, e), "abc")
	})

	t.Run("clears every register", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		e.Registers().Set('a', "one")
		e.Registers().Set('b', "two")
		assert.NoError(t, test.RunCmd(t, km, e, "clear_register").Error)
		_, ok := e.Registers().First('a')
		assert.False(t, ok)
		_, ok = e.Registers().First('b')
		assert.False(t, ok)
	})

	t.Run("clears only the named register", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		e.Registers().Set('a', "one")
		e.Registers().Set('b', "two")
		res := test.RunCmdArgs(t, km, e, "clear_register", "a")
		assert.NoError(t, res.Error)
		_, ok := e.Registers().First('a')
		assert.False(t, ok)
		kept, ok := e.Registers().First('b')
		assert.True(t, ok)
		assert.Equal(t, "two", kept)
	})

	t.Run("rejects a multi-rune name", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		res := test.RunCmdArgs(t, km, e, "clear_register", "ab")
		assert.Error(t, res.Error)
	})
}

func TestClipboardPrimarySelection(t *testing.T) {
	t.Run("yank writes the primary register", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)

		test.RunCmd(t, km, e, "yank_to_primary_clipboard")

		got, ok := e.FirstRegister(view.RegisterPrimaryClipboard)
		assert.True(t, ok)
		assert.Equal(t, "abc", got)
	})

	t.Run("paste after reads the primary register", func(t *testing.T) {
		e, km := test.Env(t, "x")
		e.WriteRegister(view.RegisterPrimaryClipboard, []string{"yz"})
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   1,
		}}, 0)

		test.RunCmd(t, km, e, "paste_primary_clipboard_after")

		assert.Equal(t, "xyz", test.DocText(t, e))
	})

	t.Run("paste before reads the primary register", func(t *testing.T) {
		e, km := test.Env(t, "z")
		e.WriteRegister(view.RegisterPrimaryClipboard, []string{"xy"})
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   1,
		}}, 0)

		test.RunCmd(t, km, e, "paste_primary_clipboard_before")

		assert.Equal(t, "xyz", test.DocText(t, e))
	})

	t.Run("replace swaps in the primary register", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		e.WriteRegister(view.RegisterPrimaryClipboard, []string{"XY"})
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 1,
			Head:   2,
		}}, 0)

		test.RunCmd(t, km, e, "primary_clipboard_paste_replace")

		assert.Equal(t, "aXYc", test.DocText(t, e))
	})
}

func TestClipboardDefaultRegister(t *testing.T) {
	t.Run("yank reaches the clipboard", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)

		test.RunCmd(t, km, e, "yank")

		got, ok := e.FirstRegister(view.RegisterClipboard)
		assert.True(t, ok)
		assert.Equal(t, "abc", got)
	})

	t.Run("delete leaves the clipboard alone", func(t *testing.T) {
		e, km := test.Env(t, "abc def")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)
		test.RunCmd(t, km, e, "yank")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 4,
			Head:   7,
		}}, 0)

		test.RunCmd(t, km, e, "delete_selection")

		clip, _ := e.FirstRegister(view.RegisterClipboard)
		assert.Equal(t, "abc", clip)
		deleted, _ := e.FirstRegister(view.RegisterDefaultYank)
		assert.Equal(t, "def", deleted)
	})

	t.Run("an explicit register wins", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)
		e.SetRegister('a')

		test.RunCmd(t, km, e, "yank")

		got, ok := e.Registers().First('a')
		assert.True(t, ok)
		assert.Equal(t, "abc", got)
		_, ok = e.FirstRegister(view.RegisterClipboard)
		assert.False(t, ok)
	})
}

func TestClipboardSystemClipboard(t *testing.T) {

	t.Run("clipboard-yank takes every selection", func(t *testing.T) {
		e, km := test.Env(t, "abc def")
		testutil.SetSelection(t, e, []core.Range{
			{Anchor: 0, Head: 3},
			{Anchor: 4, Head: 7},
		}, 0)

		test.RunCmd(t, km, e, "clipboard-yank")

		assert.Equal(t,
			[]string{"abc", "def"}, e.ReadRegister(view.RegisterClipboard),
		)
	})

	t.Run("yank main takes one selection", func(t *testing.T) {
		e, km := test.Env(t, "abc def")
		testutil.SetSelection(t, e, []core.Range{
			{Anchor: 0, Head: 3},
			{Anchor: 4, Head: 7},
		}, 0)

		test.RunCmd(t, km, e, "yank_main_selection")

		assert.Equal(t,
			[]string{"abc"}, e.ReadRegister(view.RegisterClipboard),
		)
	})

	t.Run("yank join uses the separator argument", func(t *testing.T) {
		e, km := test.Env(t, "ab")
		testutil.SetSelection(t, e, []core.Range{
			{Anchor: 0, Head: 1},
			{Anchor: 1, Head: 2},
		}, 0)

		test.RunCmdArgs(t, km, e, "yank_join", ",")

		got, ok := e.FirstRegister(view.RegisterClipboard)
		assert.True(t, ok)
		assert.Equal(t, "a,b", got)
	})

	t.Run("yank join writes the clipboard", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)

		test.RunCmd(t, km, e, "yank_join")

		got, ok := e.FirstRegister(view.RegisterClipboard)
		assert.True(t, ok)
		assert.Equal(t, "abc", got)
	})

}
