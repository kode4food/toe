package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestIndentGuides(t *testing.T) {
	t.Run("CharRune returns custom char", func(t *testing.T) {
		ig := view.IndentGuides{Character: "|"}
		assert.Equal(t, '|', ig.CharRune())
	})

	t.Run("CharRune returns default when empty", func(t *testing.T) {
		ig := view.IndentGuides{}
		assert.Equal(t, view.DefaultIndentGuideChar, ig.CharRune())
	})

	t.Run("GetSkipLevels returns 0 when nil", func(t *testing.T) {
		ig := view.IndentGuides{}
		assert.Equal(t, 0, ig.GetSkipLevels())
	})

	t.Run("GetSkipLevels returns value when set", func(t *testing.T) {
		ig := view.IndentGuides{SkipLevels: new(2)}
		assert.Equal(t, 2, ig.GetSkipLevels())
	})
}

func TestGutter(t *testing.T) {
	t.Run("default layout when not present", func(t *testing.T) {
		g := view.Gutter{}
		layout := g.GutterLayout()
		assert.Contains(t, layout, view.GutterTypeLineNumbers)
	})

	t.Run("custom layout when present", func(t *testing.T) {
		g := view.Gutter{
			Present: true,
			Layout:  []view.GutterType{view.GutterTypeLineNumbers},
		}
		assert.Equal(t,
			[]view.GutterType{view.GutterTypeLineNumbers}, g.GutterLayout())
	})

	t.Run("HasGutterType true for present type", func(t *testing.T) {
		g := view.Gutter{}
		assert.True(t, g.HasGutterType(view.GutterTypeLineNumbers))
	})

	t.Run("HasGutterType false for absent type", func(t *testing.T) {
		g := view.Gutter{
			Present: true,
			Layout:  []view.GutterType{view.GutterTypeSpacer},
		}
		assert.False(t, g.HasGutterType(view.GutterTypeLineNumbers))
	})

	t.Run("default min width when nil", func(t *testing.T) {
		g := view.Gutter{}
		assert.Equal(t,
			view.DefaultGutterLineNumberMinWidth, g.LineNumberMinWidth())
	})

	t.Run("LineNumberMinWidth returns custom value", func(t *testing.T) {
		g := view.Gutter{LineNumbers: view.GutterLineNumbers{MinWidth: new(5)}}
		assert.Equal(t, 5, g.LineNumberMinWidth())
	})

	t.Run("UnmarshalTOML with array sets layout", func(t *testing.T) {
		g := view.Gutter{}
		err := g.UnmarshalTOML([]any{"line-numbers", "spacer"})
		assert.NoError(t, err)
		assert.True(t, g.Present)
		assert.Contains(t, g.Layout, view.GutterTypeLineNumbers)
	})

	t.Run("UnmarshalTOML rejects bad type in array", func(t *testing.T) {
		g := view.Gutter{}
		err := g.UnmarshalTOML([]any{"bad-gutter-type"})
		assert.Error(t, err)
	})

	t.Run("UnmarshalTOML with map sets present", func(t *testing.T) {
		g := view.Gutter{}
		err := g.UnmarshalTOML(map[string]any{
			"layout": []any{"line-numbers"},
		})
		assert.NoError(t, err)
		assert.True(t, g.Present)
	})

	t.Run("UnmarshalTOML nil is a no-op", func(t *testing.T) {
		g := view.Gutter{}
		assert.NoError(t, g.UnmarshalTOML(nil))
		assert.False(t, g.Present)
	})

	t.Run("UnmarshalTOML rejects unknown type", func(t *testing.T) {
		g := view.Gutter{}
		assert.Error(t, g.UnmarshalTOML(42))
	})
}

func TestWhitespaceRender(t *testing.T) {
	t.Run("defaults when unset", func(t *testing.T) {
		w := view.WhitespaceRender{}
		assert.Equal(t, view.WhitespaceRenderNone, w.SpaceRender())
		assert.Equal(t, view.WhitespaceRenderNone, w.NbspRender())
		assert.Equal(t, view.WhitespaceRenderNone, w.NnbspRender())
		assert.Equal(t, view.WhitespaceRenderNone, w.TabRender())
		assert.Equal(t, view.WhitespaceRenderNone, w.NewlineRender())
	})

	t.Run("Default fills in unset specifics", func(t *testing.T) {
		w := view.WhitespaceRender{Default: new(view.WhitespaceRenderAll)}
		assert.Equal(t, view.WhitespaceRenderAll, w.SpaceRender())
		assert.Equal(t, view.WhitespaceRenderAll, w.TabRender())
	})

	t.Run("specific overrides Default", func(t *testing.T) {
		w := view.WhitespaceRender{Default: new(view.WhitespaceRenderAll), Space: new(view.WhitespaceRenderNone)}
		assert.Equal(t, view.WhitespaceRenderNone, w.SpaceRender())
		assert.Equal(t, view.WhitespaceRenderAll, w.TabRender())
	})

	t.Run("UnmarshalTOML with string sets Default", func(t *testing.T) {
		w := view.WhitespaceRender{}
		assert.NoError(t, w.UnmarshalTOML("all"))
		assert.Equal(t, view.WhitespaceRenderAll, w.SpaceRender())
	})

	t.Run("map sets per-char values", func(t *testing.T) {
		w := view.WhitespaceRender{}
		err := w.UnmarshalTOML(map[string]any{
			"space": "all", "tab": "none",
		})
		assert.NoError(t, err)
		assert.Equal(t, view.WhitespaceRenderAll, w.SpaceRender())
		assert.Equal(t, view.WhitespaceRenderNone, w.TabRender())
	})

	t.Run("UnmarshalTOML with nil is a no-op", func(t *testing.T) {
		w := view.WhitespaceRender{}
		assert.NoError(t, w.UnmarshalTOML(nil))
	})

	t.Run("UnmarshalTOML rejects invalid string", func(t *testing.T) {
		w := view.WhitespaceRender{}
		assert.Error(t, w.UnmarshalTOML("bad"))
	})

	t.Run("rejects non-string map value", func(t *testing.T) {
		w := view.WhitespaceRender{}
		assert.Error(t, w.UnmarshalTOML(map[string]any{"space": 42}))
	})
}

func TestWhitespaceCharacters(t *testing.T) {
	t.Run("rune methods return defaults when empty", func(t *testing.T) {
		w := view.WhitespaceCharacters{}
		assert.Equal(t, view.DefaultWSSpace, w.SpaceRune())
		assert.Equal(t, view.DefaultWSNbsp, w.NbspRune())
		assert.Equal(t, view.DefaultWSNnbsp, w.NnbspRune())
		assert.Equal(t, view.DefaultWSTab, w.TabRune())
		assert.Equal(t, view.DefaultWSTabpad, w.TabpadRune())
		assert.Equal(t, view.DefaultWSNewline, w.NewlineRune())
	})

	t.Run("rune methods return custom chars", func(t *testing.T) {
		w := view.WhitespaceCharacters{
			Space: "·", Tab: "→",
		}
		assert.Equal(t, '·', w.SpaceRune())
		assert.Equal(t, '→', w.TabRune())
	})
}

func TestStatusLineElement(t *testing.T) {
	t.Run("parses valid element", func(t *testing.T) {
		var s view.StatusLineElement
		assert.NoError(t, s.UnmarshalText([]byte("mode")))
		assert.Equal(t, view.StatusLineMode, s)
	})

	t.Run("rejects invalid element", func(t *testing.T) {
		var s view.StatusLineElement
		assert.Error(t, s.UnmarshalText([]byte("bad-element")))
	})
}

func TestGutterType(t *testing.T) {
	t.Run("parses valid gutter type", func(t *testing.T) {
		var g view.GutterType
		assert.NoError(t, g.UnmarshalText([]byte("line-numbers")))
		assert.Equal(t, view.GutterTypeLineNumbers, g)
	})

	t.Run("rejects invalid gutter type", func(t *testing.T) {
		var g view.GutterType
		assert.Error(t, g.UnmarshalText([]byte("bad-type")))
	})
}

func TestConfigTypes(t *testing.T) {
	t.Run("parses cursor kind", func(t *testing.T) {
		v, err := view.ParseCursorKind("bar")
		assert.NoError(t, err)
		assert.Equal(t, view.CursorKindBar, v)
	})

	t.Run("rejects cursor kind", func(t *testing.T) {
		_, err := view.ParseCursorKind("bad")
		assert.ErrorIs(t, err, view.ErrInvalidCursorKind)
	})

	t.Run("parses line number", func(t *testing.T) {
		v, err := view.ParseLineNumber("absolute")
		assert.NoError(t, err)
		assert.Equal(t, view.LineNumberAbsolute, v)
	})

	t.Run("rejects line number", func(t *testing.T) {
		_, err := view.ParseLineNumber("bad")
		assert.ErrorIs(t, err, view.ErrInvalidLineNumber)
	})

	t.Run("parses bufferline", func(t *testing.T) {
		v, err := view.ParseBufferLine("never")
		assert.NoError(t, err)
		assert.Equal(t, view.BufferLineNever, v)
	})

	t.Run("rejects bufferline", func(t *testing.T) {
		_, err := view.ParseBufferLine("bad")
		assert.ErrorIs(t, err, view.ErrInvalidBufferLine)
	})

	t.Run("parses whitespace render", func(t *testing.T) {
		v, err := view.ParseWhitespaceRenderValue("all")
		assert.NoError(t, err)
		assert.Equal(t, view.WhitespaceRenderAll, v)
	})

	t.Run("rejects whitespace render", func(t *testing.T) {
		_, err := view.ParseWhitespaceRenderValue("bad")
		assert.ErrorIs(t, err, view.ErrInvalidWhitespaceRender)
	})
}
