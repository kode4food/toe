package theme_test

import (
	"errors"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/theme"
)

func TestTheme(t *testing.T) {
	t.Run("parses style string", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": "#ffffff",
		})

		style, ok := th.TryGet("keyword")

		assert.Empty(t, warnings)
		assert.True(t, ok)
		assert.Equal(t, lipgloss.Color("#ffffff"), style.GetForeground())
	})

	t.Run("parses palette alias", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": "my-color",
			"palette": map[string]any{
				"my-color": "#ffffff",
			},
		})

		style := th.Get("keyword")

		assert.Empty(t, warnings)
		assert.Equal(t, lipgloss.Color("#ffffff"), style.GetForeground())
	})

	t.Run("invalid palette falls back to default", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": "my-color",
			"palette": map[string]any{
				"my-color": "#ffffff",
				"bad":      true,
			},
		})

		style := th.Get("keyword")

		assert.NotEmpty(t, warnings)
		assert.Equal(t, lipgloss.NoColor{}, style.GetForeground())
	})

	t.Run("parses short RGB colors", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": "short",
			"palette": map[string]any{
				"short": "#abc",
			},
			"ui.selection": "#012",
		})

		kw := th.Get("keyword")
		sel := th.Get("ui.selection")

		assert.Empty(t, warnings)
		assert.Equal(t, lipgloss.Color("#aabbcc"), kw.GetForeground())
		assert.Equal(t, lipgloss.Color("#001122"), sel.GetForeground())
		assert.False(t, th.Is16Color())
	})

	t.Run("parses style table", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"fg": "#ffffff",
				"bg": "#000000",
				"modifiers": []any{
					"bold", "italic", "slow_blink", "rapid_blink",
					"reversed", "hidden",
				},
			},
		})

		style := th.Get("keyword")

		assert.Empty(t, warnings)
		assert.Equal(t, lipgloss.Color("#ffffff"), style.GetForeground())
		assert.Equal(t, lipgloss.Color("#000000"), style.GetBackground())
		assert.True(t, style.GetBold())
		assert.True(t, style.GetItalic())
		assert.True(t, style.GetBlink())
		assert.True(t, style.GetReverse())
	})

	t.Run("parses underline table", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"diagnostic.error": map[string]any{
				"underline": map[string]any{
					"color": "#ff0000",
					"style": "curl",
				},
			},
		})

		style := th.Get("diagnostic.error")

		assert.Empty(t, warnings)
		assert.Equal(t, lipgloss.UnderlineCurly, style.GetUnderlineStyle())
		assert.Equal(t, lipgloss.Color("#ff0000"), style.GetUnderlineColor())
	})

	t.Run("underline color implies line", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"diagnostic.warning": map[string]any{
				"underline": map[string]any{
					"color": "#ffaa00",
				},
			},
		})

		style := th.Get("diagnostic.warning")

		assert.Empty(t, warnings)
		assert.True(t, style.GetUnderline())
		assert.Equal(t, lipgloss.Color("#ffaa00"), style.GetUnderlineColor())
	})

	t.Run("invalid underline warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"diagnostic.info": map[string]any{
				"underline": map[string]any{
					"bogus": true,
				},
			},
		})

		assert.NotEmpty(t, warnings)
	})

	t.Run("falls back through dot scopes", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"ui.text": "white",
		})

		style, ok := th.TryGet("ui.text.focus")
		_, exact := th.TryGetExact("ui.text.focus")

		assert.Empty(t, warnings)
		assert.True(t, ok)
		assert.False(t, exact)
		assert.Equal(t, lipgloss.Color("15"), style.GetForeground())
	})

	t.Run("missing scope returns false", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"ui.text": "white",
		})

		_, ok := th.TryGet("ui.popup")

		assert.Empty(t, warnings)
		assert.False(t, ok)
	})

	t.Run("detects 16 color theme", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"ui.text": "white",
			"ui.selection": map[string]any{
				"bg": "1",
			},
		})

		assert.Empty(t, warnings)
		assert.True(t, th.Is16Color())
	})

	t.Run("adds default rainbow scopes", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"ui.selection": "white",
		})

		style, ok := th.TryGetExact("rainbow.0")

		assert.Empty(t, warnings)
		assert.True(t, ok)
		assert.Equal(t, lipgloss.Color("1"), style.GetForeground())
		assert.Equal(t, 6, th.RainbowLength())
		assert.Len(t, th.Scopes(), 7)
	})

	t.Run("parses custom rainbow styles", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"rainbow": []any{
				"#010203",
				map[string]any{"fg": "yellow"},
			},
			"ui.selection": "white",
		})

		first := th.Get("rainbow.0")
		second := th.Get("rainbow.1")
		_, old := th.TryGetExact("rainbow")

		assert.Empty(t, warnings)
		assert.False(t, old)
		assert.Equal(t, lipgloss.Color("#010203"), first.GetForeground())
		assert.Equal(t, lipgloss.Color("3"), second.GetForeground())
		assert.Equal(t, 2, th.RainbowLength())
		assert.False(t, th.Is16Color())
	})

	t.Run("invalid rainbow falls back to default", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"rainbow":      "bad",
			"ui.selection": "white",
		})

		style := th.Get("rainbow.0")

		assert.NotEmpty(t, warnings)
		assert.Equal(t, lipgloss.Color("1"), style.GetForeground())
	})

	t.Run("detects RGB foreground and background", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"ui.text": "#ffffff",
			"ui.selection": map[string]any{
				"bg": "#000000",
			},
		})

		assert.Empty(t, warnings)
		assert.False(t, th.Is16Color())
	})

	t.Run("detects RGB palette aliases", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"palette": map[string]any{
				"surface": "#000000",
			},
			"ui.selection": map[string]any{
				"bg": "surface",
			},
		})

		assert.Empty(t, warnings)
		assert.False(t, th.Is16Color())
	})

	t.Run("underline ignored for 16 color", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"ui.selection": map[string]any{
				"underline": map[string]any{
					"color": "#ffffff",
					"style": "line",
				},
			},
		})

		assert.Empty(t, warnings)
		assert.True(t, th.Is16Color())
	})

	t.Run("decodes built-in mocha", func(t *testing.T) {
		data, err := loader.LoadThemeTOML("mocha")
		assert.NoError(t, err)

		th, warnings := theme.Decode(data)
		style, ok := th.TryGet("ui.text.focus")

		assert.Empty(t, warnings)
		assert.True(t, ok)
		assert.Equal(t, lipgloss.Color("#cdd6f4"), style.GetForeground())
		assert.NotEmpty(t, th.Scopes())
	})
}

func TestLoad(t *testing.T) {
	t.Run("loads and decodes built-in theme", func(t *testing.T) {
		th, warnings, err := theme.Load("mocha")

		style, ok := th.TryGet("ui.text")

		assert.NoError(t, err)
		assert.Empty(t, warnings)
		assert.Equal(t, "mocha", th.Name())
		assert.True(t, ok)
		assert.Equal(t, lipgloss.Color("#cdd6f4"), style.GetForeground())
	})

	t.Run("rejects unsupported theme", func(t *testing.T) {
		_, _, err := theme.Load("bad")

		assert.True(t, errors.Is(err, loader.ErrThemeNotFound))
	})
}

func TestThemeDefault(t *testing.T) {
	t.Run("Default loads mocha theme", func(t *testing.T) {
		th, warnings, err := theme.Default()
		assert.NoError(t, err)
		assert.NotNil(t, th)
		assert.Equal(t, "mocha", th.Name())
		_ = warnings
	})
}

func TestThemeValidate(t *testing.T) {
	t.Run("theme missing ui.selection fails", func(t *testing.T) {
		th, _ := theme.Decode(map[string]any{
			"keyword": "#ffffff",
		})
		assert.Error(t, th.Validate())
	})

	t.Run("theme with ui.selection passes", func(t *testing.T) {
		th, _ := theme.Decode(map[string]any{
			"ui.selection": "#ffffff",
		})
		assert.NoError(t, th.Validate())
	})
}

func TestUnderlineStyles(t *testing.T) {
	for _, tc := range []struct {
		name  string
		style string
		ok    bool
	}{
		{"line", "line", true},
		{"curl", "curl", true},
		{"dashed", "dashed", true},
		{"dotted", "dotted", true},
		{"double_line", "double_line", true},
		{"invalid", "squiggly", false},
		{"non-string", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var value any = tc.style
			if tc.name == "non-string" {
				value = 42
			}
			th, warnings := theme.Decode(map[string]any{
				"ui.selection": "#ffffff",
				"keyword": map[string]any{
					"underline": map[string]any{
						"style": value,
					},
				},
			})
			if tc.ok {
				assert.Empty(t, warnings)
				_, ok := th.TryGet("keyword")
				assert.True(t, ok)
			} else {
				assert.NotEmpty(t, warnings)
			}
		})
	}
}

func TestModifiers(t *testing.T) {
	t.Run("dim modifier", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"modifiers": []any{"dim"},
			},
		})
		assert.Empty(t, warnings)
		style := th.Get("keyword")
		assert.True(t, style.GetFaint())
	})

	t.Run("underlined modifier", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"modifiers": []any{"underlined"},
			},
		})
		assert.Empty(t, warnings)
		style := th.Get("keyword")
		assert.True(t, style.GetUnderline())
	})

	t.Run("crossed_out modifier", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"modifiers": []any{"crossed_out"},
			},
		})
		assert.Empty(t, warnings)
		style := th.Get("keyword")
		assert.True(t, style.GetStrikethrough())
	})

	t.Run("hidden modifier skipped", func(t *testing.T) {
		th, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"modifiers": []any{"hidden"},
			},
		})
		assert.Empty(t, warnings)
		_ = th.Get("keyword")
	})

	t.Run("invalid modifier name warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"modifiers": []any{"blinking"},
			},
		})
		assert.NotEmpty(t, warnings)
	})

	t.Run("non-string modifier warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"modifiers": []any{42},
			},
		})
		assert.NotEmpty(t, warnings)
	})

	t.Run("non-array modifiers warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"modifiers": "bold",
			},
		})
		assert.NotEmpty(t, warnings)
	})
}

func TestUnderlineInvalidAttr(t *testing.T) {
	t.Run("unknown underline attribute warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"keyword": map[string]any{
				"underline": map[string]any{
					"thickness": "2px",
				},
			},
		})
		assert.NotEmpty(t, warnings)
	})
}

func TestRawColorEdgeCases(t *testing.T) {
	t.Run("malformed hex length warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"keyword": "#ff00",
		})
		assert.NotEmpty(t, warnings)
	})

	t.Run("malformed hex content warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"keyword": "#xxyyzz",
		})
		assert.NotEmpty(t, warnings)
	})

	t.Run("ansi index out of range warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"keyword": "300",
		})
		assert.NotEmpty(t, warnings)
	})

	t.Run("non-string color value warns", func(t *testing.T) {
		_, warnings := theme.Decode(map[string]any{
			"keyword": 42,
		})
		assert.NotEmpty(t, warnings)
	})
}
