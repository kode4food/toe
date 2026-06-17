package ui

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
)

// Context holds shared mutable state accessible to all compositor layers
type Context struct {
	Editor  *view.Editor
	Keymaps *command.Keymaps

	lastLayer func(*view.Editor) layerFunc

	theme     *theme.Theme
	themeName string
}

// Theme returns the active theme, reloading it when the configured theme name
// has changed. Falls back to the embedded default if the configured theme
// fails to load; the embedded default is guaranteed to succeed
func (c *Context) Theme() *theme.Theme {
	name := c.Editor.Config().Theme.Choose(false)
	if name == c.themeName {
		return c.theme
	}
	c.themeName = name
	if th, _, err := theme.Load(name); err == nil {
		c.theme = th
		return c.theme
	}
	th, _, err := theme.Default()
	if err != nil {
		panic("embedded default theme failed to load: " + err.Error())
	}
	c.theme = th
	return c.theme
}
