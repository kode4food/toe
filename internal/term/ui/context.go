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

	pickerLayout PickerLayoutOptions
	loadedTheme  *theme.Theme
	theme        string
}

// Theme returns the active theme, reloading it when the configured theme name
// has changed. Falls back to the embedded default if the configured theme
// fails to load. If the embedded default is unavailable, returns a minimal
// decoded theme so rendering can continue
func (c *Context) Theme() *theme.Theme {
	name := c.Editor.Options().Theme
	if name == c.theme {
		return c.loadedTheme
	}
	c.theme = name
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
