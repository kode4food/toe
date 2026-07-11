package ui

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
)

// Context holds shared mutable state accessible to all compositor layers
type Context struct {
	Editor  *view.Editor
	Keymaps *command.Keymaps
	Syntax  *syntax.Cache

	SingleLayer bool
	lastLayer   func(*view.Editor) layerFunc

	OverlayRegions        []Bounds
	OverlayRegionsPrecise bool
	OverlaysChanged       bool

	pickerLayout PickerLayoutOptions
	loadedTheme  *theme.Theme
	theme        string
	styleGen     int
}

// StyleGen returns a counter that increments whenever the active theme changes,
// letting cached overlay buffers know they must repaint even without their own
// content changing
func (c *Context) StyleGen() int {
	return c.styleGen
}

// Theme returns the active theme, reloading it if the configured name
// changed, falling back to the embedded default on load failure
func (c *Context) Theme() *theme.Theme {
	name := c.Editor.Options().Theme
	if name == c.theme {
		return c.loadedTheme
	}
	c.theme = name
	c.styleGen++
	if th, _, err := theme.Load(name); err == nil {
		c.loadedTheme = th
		return c.loadedTheme
	}
	th, _, err := theme.Default()
	if err != nil {
		c.loadedTheme = fallbackTheme()
		return c.loadedTheme
	}
	c.loadedTheme = th
	return c.loadedTheme
}

func fallbackTheme() *theme.Theme {
	th, _ := theme.Decode(map[string]any{
		"ui.selection": "default",
	})
	return th
}
