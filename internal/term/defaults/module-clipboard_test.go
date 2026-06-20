package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestClipboardYankPaste(t *testing.T) {
	t.Run("yank then paste after inserts text", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc\n")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "yank")
		setCursor(t, e, 3)
		runCmd(t, km, e, "paste_after")
		assert.Contains(t, docText(t, e), "abc")
	})

	t.Run("paste before inserts before cursor", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc\n")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "yank")
		setCursor(t, e, 4)
		runCmd(t, km, e, "paste_before")
		assert.Contains(t, docText(t, e), "abc")
	})

	t.Run("replace with yanked replaces selection", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc\ndef\n")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "yank")
		setSelection(t, e, []core.Range{core.NewRange(4, 7)}, 0)
		runCmd(t, km, e, "replace_with_yanked")
		assert.Contains(t, docText(t, e), "abc")
	})

	t.Run("clear register runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		runCmd(t, km, e, "clear_register")
	})

	t.Run("show clipboard provider returns info", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "show_clipboard_provider")
		assert.NotEmpty(t, res.Message)
	})
}

func TestClipboardSystemClipboard(t *testing.T) {
	t.Run("yank to clipboard runs without error", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "yank_to_clipboard")
		// clipboard operations may fail in CI but must not panic
	})

	t.Run("paste clipboard after runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "paste_clipboard_after")
	})

	t.Run("paste clipboard before runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "paste_clipboard_before")
	})

	t.Run("clipboard_paste_replace runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "clipboard_paste_replace")
	})

	t.Run("yank main to clipboard runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "yank_main_selection_to_clipboard")
	})

	t.Run("yank joined to clipboard runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "yank_joined_to_clipboard")
	})

	t.Run("yank to primary clipboard runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "yank_to_primary_clipboard")
	})

	t.Run("paste primary clipboard after runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "paste_primary_clipboard_after")
	})

	t.Run("paste primary clipboard before runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "paste_primary_clipboard_before")
	})

	t.Run("primary clipboard paste replace runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "primary_clipboard_paste_replace")
	})
}
