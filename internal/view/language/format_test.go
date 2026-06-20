package language_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/language"
)

func TestDefaultTextFormat(t *testing.T) {
	t.Run("sets viewport width", func(t *testing.T) {
		f := language.DefaultTextFormat(80)
		assert.Equal(t, 80, f.ViewportWidth)
	})

	t.Run("has default tab width", func(t *testing.T) {
		f := language.DefaultTextFormat(100)
		assert.Equal(t, language.DefaultTabWidth, f.TabWidth)
		assert.False(t, f.SoftWrap)
	})

	t.Run("caps max wrap at viewport/4", func(t *testing.T) {
		f := language.DefaultTextFormat(40)
		assert.Equal(t, 10, f.MaxWrap)
	})
}

func TestTextFormatForConfig(t *testing.T) {
	t.Run("soft wrap disabled by default", func(t *testing.T) {
		f := language.TextFormatForConfig(
			&language.Language{}, nil, language.SoftWrap{}, 80,
		)
		assert.False(t, f.SoftWrap)
	})

	t.Run("soft wrap enabled via editor config", func(t *testing.T) {
		sw := language.SoftWrap{Enable: new(true)}
		f := language.TextFormatForConfig(
			&language.Language{}, nil, sw, 80,
		)
		assert.True(t, f.SoftWrap)
	})

	t.Run("language enable overrides editor", func(t *testing.T) {
		lang := &language.Language{
			SoftWrap: language.SoftWrap{Enable: new(true)},
		}
		f := language.TextFormatForConfig(
			lang, nil, language.SoftWrap{}, 80,
		)
		assert.True(t, f.SoftWrap)
	})

	t.Run("custom wrap indicator from language", func(t *testing.T) {
		lang := &language.Language{
			SoftWrap: language.SoftWrap{WrapIndicator: new(">> ")},
		}
		f := language.TextFormatForConfig(
			lang, nil, language.SoftWrap{}, 80,
		)
		assert.Equal(t, ">> ", f.WrapIndicator)
	})

	t.Run("wrap at text width narrows viewport", func(t *testing.T) {
		lang := &language.Language{
			TextWidth: new(40),
			SoftWrap:  language.SoftWrap{WrapAtTextWidth: new(true)},
		}
		f := language.TextFormatForConfig(
			lang, nil, language.SoftWrap{}, 80,
		)
		assert.True(t, f.SoftWrapAtTextWidth)
		assert.Equal(t, 40, f.ViewportWidth)
	})

	t.Run("large text width disables wrap", func(t *testing.T) {
		lang := &language.Language{
			TextWidth: new(100),
			SoftWrap:  language.SoftWrap{WrapAtTextWidth: new(true)},
		}
		f := language.TextFormatForConfig(
			lang, nil, language.SoftWrap{}, 80,
		)
		assert.False(t, f.SoftWrapAtTextWidth)
	})

	t.Run("max wrap from language", func(t *testing.T) {
		lang := &language.Language{
			SoftWrap: language.SoftWrap{MaxWrap: new(5)},
		}
		f := language.TextFormatForConfig(
			lang, nil, language.SoftWrap{}, 80,
		)
		assert.Equal(t, 5, f.MaxWrap)
	})

	t.Run("max indent retain from language", func(t *testing.T) {
		lang := &language.Language{
			SoftWrap: language.SoftWrap{MaxIndentRetain: new(10)},
		}
		f := language.TextFormatForConfig(
			lang, nil, language.SoftWrap{}, 80,
		)
		assert.Equal(t, 10, f.MaxIndentRetain)
	})

	t.Run("text width from language", func(t *testing.T) {
		lang := &language.Language{TextWidth: new(60)}
		f := language.TextFormatForConfig(
			lang, nil, language.SoftWrap{}, 80,
		)
		assert.Equal(t, language.DefaultTextWidth, 80)
		_ = f
	})

	t.Run("text width from editor config", func(t *testing.T) {
		f := language.TextFormatForConfig(
			&language.Language{}, new(72), language.SoftWrap{}, 80,
		)
		_ = f
	})
}

func TestTextFormatForLanguageWithConfig(t *testing.T) {
	t.Run("unknown language returns default format", func(t *testing.T) {
		f := language.TextFormatForLanguageWithConfig(
			"no-such-lang", nil, language.SoftWrap{}, 80,
		)
		assert.NotNil(t, f)
		assert.Equal(t, 80, f.ViewportWidth)
	})
}
