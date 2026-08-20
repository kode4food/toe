package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/tui"
)

func TestContext(t *testing.T) {
	t.Run("dimmed theme follows InactiveDim", func(t *testing.T) {
		e := editorWithText(t, "hello toe")
		e.Options().InactiveDim = 10
		cx := &ui.Context{Editor: e}
		_ = cx.ThemeFor(false)

		e.Options().InactiveDim = 50
		dimmed := cx.ThemeFor(false)

		// mocha ui.background (30,30,46) darkened 50%. A theme-name change is
		// not required for a changed InactiveDim to take effect
		assert.Equal(t,
			tui.ColorRGB(15, 15, 23), dimmed.Get("ui.background").BgColor(),
		)
	})
}
