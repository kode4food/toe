package ui

import (
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Context holds shared mutable state accessible to all compositor layers
	Context struct {
		Editor  *view.Editor
		Keymaps *command.Keymaps
		Syntax  *syntax.Cache

		composition compositionState
		theme       themeState

		lastLayer    func(*view.Editor) layerFunc
		pickerLayout PickerLayoutOptions
		images       *imageRegistry
	}

	compositionState struct {
		singleLayer bool
		regions     []geom.Area
		precise     bool
		changed     bool
	}

	themeState struct {
		loaded     *theme.Theme
		name       string
		generation int
	}
)

// StyleGen returns a counter that increments whenever the active theme changes,
// letting cached overlay buffers know they must repaint even without their own
// content changing
func (c *Context) StyleGen() int {
	return c.theme.generation
}

// Theme returns the active theme, reloading it if the configured name
// changed, falling back to the embedded default on load failure
func (c *Context) Theme() *theme.Theme {
	name := c.Editor.Options().Theme
	if name == c.theme.name {
		return c.theme.loaded
	}
	c.theme.name = name
	c.theme.generation++
	if th, _, err := theme.Load(name); err == nil {
		c.theme.loaded = th
		return c.theme.loaded
	}
	th, _, err := theme.Default()
	if err != nil {
		c.theme.loaded = fallbackTheme()
		return c.theme.loaded
	}
	c.theme.loaded = th
	return c.theme.loaded
}

func fallbackTheme() *theme.Theme {
	th, _ := theme.Decode(map[string]any{
		"ui.selection": "default",
	})
	return th
}
