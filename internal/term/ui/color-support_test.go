package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/ui"
)

var colorEnvKeys = []string{
	"COLORTERM", "TERM", "TERM_PROGRAM", "VTE_VERSION",
	"WSL_DISTRO_NAME", "KITTY_WINDOW_ID", "WEZTERM_EXECUTABLE",
	"ALACRITTY_WINDOW_ID", "KONSOLE_VERSION", "WT_SESSION",
}

func TestTrueColorSupported(t *testing.T) {
	t.Run("bare terminal is unsupported", func(t *testing.T) {
		clearColorEnv(t)
		assert.False(t, ui.TrueColorSupported())
	})

	t.Run("256-color term is unsupported", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("TERM", "xterm-256color")
		assert.False(t, ui.TrueColorSupported())
	})

	t.Run("colorterm truecolor", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("COLORTERM", "truecolor")
		assert.True(t, ui.TrueColorSupported())
	})

	t.Run("colorterm 24bit", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("COLORTERM", "24bit")
		assert.True(t, ui.TrueColorSupported())
	})

	t.Run("colorterm other value", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("COLORTERM", "8bit")
		assert.False(t, ui.TrueColorSupported())
	})

	t.Run("ssh keeps term_program", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("TERM", "xterm-256color")
		t.Setenv("TERM_PROGRAM", "ghostty")
		assert.True(t, ui.TrueColorSupported())
	})

	t.Run("direct terminfo suffix", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("TERM", "xterm-direct")
		assert.True(t, ui.TrueColorSupported())
	})

	t.Run("terminal name in term", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("TERM", "xterm-ghostty")
		assert.True(t, ui.TrueColorSupported())
	})

	t.Run("presence-only signal", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("KITTY_WINDOW_ID", "3")
		assert.True(t, ui.TrueColorSupported())
	})

	t.Run("vte at threshold", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("VTE_VERSION", "3600")
		assert.True(t, ui.TrueColorSupported())
	})

	t.Run("vte below threshold", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("VTE_VERSION", "3409")
		assert.False(t, ui.TrueColorSupported())
	})

	t.Run("vte garbage value", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("VTE_VERSION", "not-a-number")
		assert.False(t, ui.TrueColorSupported())
	})
}

func clearColorEnv(t *testing.T) {
	t.Helper()
	for _, k := range colorEnvKeys {
		t.Setenv(k, "")
	}
}
