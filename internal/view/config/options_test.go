package config_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/config"
)

func TestOptions(t *testing.T) {
	t.Run("gets modeled option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		value, err := config.GetOption(cfg, "editor.text-width")

		assert.NoError(t, err)
		assert.Equal(t, "80", value)
	})

	t.Run("sets modeled option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		err := config.SetOption(cfg, "editor.text-width", "72")

		assert.NoError(t, err)
		assert.Equal(t, 72, *cfg.Editor.TextWidth)
	})

	t.Run("sets quoted string option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		err := config.SetOption(
			cfg, "editor.soft-wrap.wrap-indicator", "'» '",
		)

		assert.NoError(t, err)
		assert.Equal(t, "» ", *cfg.Editor.SoftWrap.WrapIndicator)
	})

	t.Run("sets array options", func(t *testing.T) {
		cfg := config.DefaultConfig()

		rulersErr := config.SetOption(
			cfg, "editor.rulers", "[80, 120]",
		)
		shellErr := config.SetOption(
			cfg, "editor.shell", `["bash", "--norc", "-c"]`,
		)

		assert.NoError(t, rulersErr)
		assert.NoError(t, shellErr)
		assert.Equal(t, []int{80, 120}, cfg.Rulers())
		assert.Equal(t, []string{"bash", "--norc", "-c"}, cfg.Shell())
	})

	t.Run("toggles boolean option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		value, err := config.ToggleOption(cfg, "editor.continue-comments")

		assert.NoError(t, err)
		assert.Equal(t, "false", value)
		assert.False(t, *cfg.Editor.ContinueComments)
	})

	t.Run("auto pair option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		err := config.SetOption(cfg, "editor.auto-pairs", "false")
		value, getErr := config.GetOption(cfg, "editor.auto-pairs")
		toggled, toggleErr := config.ToggleOption(
			cfg, "editor.auto-pairs",
		)

		assert.NoError(t, err)
		assert.NoError(t, getErr)
		assert.Equal(t, "false", value)
		assert.NoError(t, toggleErr)
		assert.Equal(t, "true", toggled)
		_, ok := cfg.AutoPairs()
		assert.True(t, ok)
	})

	t.Run("auto save options", func(t *testing.T) {
		cfg := config.DefaultConfig()

		err := config.SetOption(
			cfg, "editor.auto-save.after-delay.timeout", "1200",
		)
		value, toggleErr := config.ToggleOption(
			cfg, "editor.auto-save.after-delay.enable",
		)
		focus, focusErr := config.ToggleOption(
			cfg, "editor.auto-save.focus-lost",
		)
		timeout, getErr := config.GetOption(
			cfg, "editor.auto-save.after-delay.timeout",
		)

		assert.NoError(t, err)
		assert.NoError(t, toggleErr)
		assert.Equal(t, "true", value)
		assert.NoError(t, focusErr)
		assert.Equal(t, "true", focus)
		assert.NoError(t, getErr)
		assert.Equal(t, "1200", timeout)
		assert.True(t, cfg.AutoSaveAfterDelay())
		assert.True(t, cfg.AutoSaveFocusLost())
	})

	t.Run("cursor shape options", func(t *testing.T) {
		cfg := config.DefaultConfig()

		err := config.SetOption(
			cfg, "editor.cursor-shape.insert", "bar",
		)
		value, getErr := config.GetOption(
			cfg, "editor.cursor-shape.insert",
		)

		assert.NoError(t, err)
		assert.NoError(t, getErr)
		assert.Equal(t, "bar", value)
		assert.Equal(t, config.CursorKindBar,
			cfg.CursorShapeForMode("insert"),
		)
	})

	t.Run("statusline mode options", func(t *testing.T) {
		cfg := config.DefaultConfig()

		sepErr := config.SetOption(
			cfg, "editor.statusline.separator", "|",
		)
		err := config.SetOption(
			cfg, "editor.statusline.mode.normal", "NORMAL",
		)
		sep, sepGetErr := config.GetOption(
			cfg, "editor.statusline.separator",
		)
		value, getErr := config.GetOption(
			cfg, "editor.statusline.mode.normal",
		)

		assert.NoError(t, sepErr)
		assert.NoError(t, err)
		assert.NoError(t, sepGetErr)
		assert.NoError(t, getErr)
		assert.Equal(t, "|", sep)
		assert.Equal(t, "NORMAL", value)
		assert.Equal(t, "|", cfg.StatusLineSeparator())
		assert.Equal(t, "NORMAL", cfg.ModeNameForMode("normal"))
	})

	t.Run("search options", func(t *testing.T) {
		cfg := config.DefaultConfig()

		smart, smartErr := config.ToggleOption(
			cfg, "editor.search.smart-case",
		)
		wrap, wrapErr := config.ToggleOption(
			cfg, "editor.search.wrap-around",
		)

		assert.NoError(t, smartErr)
		assert.Equal(t, "false", smart)
		assert.NoError(t, wrapErr)
		assert.Equal(t, "false", wrap)
		assert.False(t, cfg.SearchSmartCase())
		assert.False(t, cfg.SearchWrapAround())
	})

	t.Run("scrolloff option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		value, getErr := config.GetOption(cfg, "editor.scrolloff")
		setErr := config.SetOption(cfg, "editor.scrolloff", "3")
		set, setGetErr := config.GetOption(cfg, "editor.scrolloff")

		assert.NoError(t, getErr)
		assert.Equal(t, "5", value)
		assert.NoError(t, setErr)
		assert.NoError(t, setGetErr)
		assert.Equal(t, "3", set)
		assert.Equal(t, 3, cfg.Scrolloff())
	})

	t.Run("line-number option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		value, getErr := config.GetOption(cfg, "editor.line-number")
		setErr := config.SetOption(
			cfg, "editor.line-number", "relative",
		)
		set, setGetErr := config.GetOption(cfg, "editor.line-number")

		assert.NoError(t, getErr)
		assert.Equal(t, "absolute", value)
		assert.NoError(t, setErr)
		assert.NoError(t, setGetErr)
		assert.Equal(t, "relative", set)
		assert.Equal(t, config.LineNumberRelative, cfg.LineNumber())
	})

	t.Run("cursorline option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		value, getErr := config.GetOption(cfg, "editor.cursorline")
		toggleErr := config.SetOption(cfg, "editor.cursorline", "true")
		set, setGetErr := config.GetOption(cfg, "editor.cursorline")
		toggled, toggledErr := config.ToggleOption(
			cfg, "editor.cursorline",
		)

		assert.NoError(t, getErr)
		assert.Equal(t, "false", value)
		assert.NoError(t, toggleErr)
		assert.NoError(t, setGetErr)
		assert.Equal(t, "true", set)
		assert.NoError(t, toggledErr)
		assert.Equal(t, "false", toggled)
		assert.False(t, cfg.Cursorline())
	})

	t.Run("whitespace render option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		value, getErr := config.GetOption(cfg, "editor.whitespace.render")
		setErr := config.SetOption(cfg, "editor.whitespace.render", "all")
		set, setGetErr := config.GetOption(cfg, "editor.whitespace.render")
		resetErr := config.SetOption(cfg, "editor.whitespace.render", "none")
		none, noneGetErr := config.GetOption(cfg, "editor.whitespace.render")

		assert.NoError(t, getErr)
		assert.Equal(t, "none", value)
		assert.NoError(t, setErr)
		assert.NoError(t, setGetErr)
		assert.Equal(t, "all", set)
		assert.NoError(t, resetErr)
		assert.NoError(t, noneGetErr)
		assert.Equal(t, "none", none)
		assert.Equal(t, config.WhitespaceRenderNone,
			cfg.Editor.Whitespace.Render.SpaceRender(),
		)
	})

	t.Run("gutters line-numbers min-width option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		value, getErr := config.GetOption(
			cfg, "editor.gutters.line-numbers.min-width",
		)
		setErr := config.SetOption(
			cfg, "editor.gutters.line-numbers.min-width", "5",
		)
		set, setGetErr := config.GetOption(
			cfg, "editor.gutters.line-numbers.min-width",
		)

		assert.NoError(t, getErr)
		assert.Equal(t, "3", value)
		assert.NoError(t, setErr)
		assert.NoError(t, setGetErr)
		assert.Equal(t, "5", set)
		g := cfg.Gutters()
		assert.Equal(t, 5, g.LineNumberMinWidth())
	})

	t.Run("indent-guides options", func(t *testing.T) {
		cfg := config.DefaultConfig()

		render, renderErr := config.GetOption(cfg, "editor.indent-guides.render")
		_, toggleErr := config.ToggleOption(cfg, "editor.indent-guides.render")
		skipErr := config.SetOption(
			cfg, "editor.indent-guides.skip-levels", "2",
		)
		skip, skipGetErr := config.GetOption(
			cfg, "editor.indent-guides.skip-levels",
		)
		charErr := config.SetOption(
			cfg, "editor.indent-guides.character", "┊",
		)
		char, charGetErr := config.GetOption(
			cfg, "editor.indent-guides.character",
		)

		assert.NoError(t, renderErr)
		assert.Equal(t, "false", render)
		assert.NoError(t, toggleErr)
		assert.True(t, cfg.Editor.IndentGuides.Render)
		assert.NoError(t, skipErr)
		assert.NoError(t, skipGetErr)
		assert.Equal(t, "2", skip)
		assert.Equal(t, 2, cfg.IndentGuides().GetSkipLevels())
		assert.NoError(t, charErr)
		assert.NoError(t, charGetErr)
		assert.Equal(t, "┊", char)
		assert.Equal(t, '┊', cfg.IndentGuides().CharRune())
	})

	t.Run("bufferline option", func(t *testing.T) {
		cfg := config.DefaultConfig()

		bl, blErr := config.GetOption(cfg, "editor.bufferline")
		setBlErr := config.SetOption(cfg, "editor.bufferline", "multiple")

		assert.NoError(t, blErr)
		assert.Equal(t, "never", bl)
		assert.NoError(t, setBlErr)
		assert.Equal(t, config.BufferLineMultiple, cfg.GetBufferLine())
	})

	t.Run("misc editor options", func(t *testing.T) {
		cfg := config.DefaultConfig()

		mouse, mouseErr := config.GetOption(cfg, "editor.mouse")
		_, toggleMouseErr := config.ToggleOption(cfg, "editor.mouse")
		middle, middleErr := config.GetOption(
			cfg, "editor.middle-click-paste",
		)
		_, toggleMiddleErr := config.ToggleOption(
			cfg, "editor.middle-click-paste",
		)

		assert.NoError(t, mouseErr)
		assert.Equal(t, "true", mouse)
		assert.NoError(t, toggleMouseErr)
		assert.False(t, cfg.Mouse())
		assert.NoError(t, middleErr)
		assert.Equal(t, "true", middle)
		assert.NoError(t, toggleMiddleErr)
		assert.False(t, cfg.MiddleClickPaste())
	})

	t.Run("rejects unknown option", func(t *testing.T) {
		_, err := config.GetOption(config.DefaultConfig(), "editor.unknown")

		assert.True(t, errors.Is(err, config.ErrUnknownOption))
	})
}

func TestOptionKeys(t *testing.T) {
	t.Run("every key is gettable", func(t *testing.T) {
		cfg := config.DefaultConfig()
		for _, key := range config.OptionKeys() {
			_, err := config.GetOption(cfg, key)
			assert.NoError(t, err)
		}
	})

	t.Run("bool keys are a subset of all keys", func(t *testing.T) {
		all := make(map[string]bool, len(config.OptionKeys()))
		for _, k := range config.OptionKeys() {
			all[k] = true
		}
		for _, k := range config.BoolOptionKeys() {
			assert.True(t, all[k])
		}
	})

	t.Run("every bool key is toggleable", func(t *testing.T) {
		cfg := config.DefaultConfig()
		for _, key := range config.BoolOptionKeys() {
			_, err := config.ToggleOption(cfg, key)
			assert.NoError(t, err)
		}
	})
}
