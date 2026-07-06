package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

func TestPreviewRender(t *testing.T) {
	t.Run("does not render rulers", func(t *testing.T) {
		th, _, err := theme.Load("mocha")
		assert.NoError(t, err)
		buf := tui.NewBuffer(24, 1)
		opts := &view.Options{Rulers: []int{10}}
		args := &previewDocRender{
			text: core.NewRope("package main\n"),
			format: &language.TextFormat{
				ViewportWidth: 24,
				TabWidth:      language.DefaultTabWidth,
			},
			opts: opts, th: th, w: 24, h: 1,
			hlFrom: 0, hlTo: 0,
		}

		renderPreviewDocInto(buf, 0, 0, args)

		hlBg := lipglossToTUIStyle(th.Get("ui.highlight")).BgColor()
		rulerBg := lipglossToTUIStyle(th.Get("ui.virtual.ruler")).BgColor()
		got := buf.Get(9, 0).Style.BgColor()

		assert.Equal(t, hlBg, got)
		assert.NotEqual(t, rulerBg, got)
	})
}
