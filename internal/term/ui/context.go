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

		windowTitle  string
		lastLayer    func(*view.Editor) layerFunc
		pickerLayout PickerLayoutOptions
		images       *imageRegistry
		fileWatcher  *fileWatcher
	}

	compositionState struct {
		singleLayer bool
		regions     []geom.Area
		precise     bool
		changed     bool
	}

	themeState struct {
		active     *theme.Theme
		dimmed     *theme.Theme
		name       string
		dim        int
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
	c.ensureTheme()
	return c.theme.active
}

// ThemeFor returns the active theme, or its dimmed variant for an unfocused
// pane
func (c *Context) ThemeFor(focused bool) *theme.Theme {
	c.ensureTheme()
	if focused {
		return c.theme.active
	}
	return c.theme.dimmed
}

func (c *Context) ensureTheme() {
	name := c.Editor.Options().Theme
	dim := min(max(c.Editor.Options().InactiveDim, 0), 90)
	if name == c.theme.name && dim == c.theme.dim {
		return
	}
	if name != c.theme.name {
		c.theme.name = name
		c.theme.active = paletteFor(loadTheme(name))
	}
	c.theme.dim = dim
	c.theme.dimmed = paletteFor(c.theme.active.Dimmed(100 - dim))
	c.theme.generation++
}

func loadTheme(name string) *theme.Theme {
	if th, err := theme.Load(name); err == nil {
		return th
	}
	if th, err := theme.Default(); err == nil {
		return th
	}
	return fallbackTheme()
}

func paletteFor(th *theme.Theme) *theme.Theme {
	if TrueColorSupported() {
		return th
	}
	return th.Quantized()
}

func fallbackTheme() *theme.Theme {
	th, _ := theme.Decode(map[string]any{
		"ui.selection": "default",
	})
	return th
}
