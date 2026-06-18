package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestModelLifecycle(t *testing.T) {
	newModel := func() ui.Model {
		e := view.NewEditor(t.TempDir())
		return resize(ui.New(e, command.NewKeymaps()), 80, 24)
	}

	t.Run("init produces a renderable model", func(t *testing.T) {
		m := newModel()
		_ = m.Init()
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("with startup cmd renders", func(t *testing.T) {
		m := newModel().WithStartupCmd(nil)
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("with initial picker renders", func(t *testing.T) {
		m := newModel().WithInitialPicker(ui.FilePicker)
		_ = m.Init()
		assert.NotEmpty(t, m.View().Content)
	})
}
