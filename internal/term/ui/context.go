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
		picker      pickerState
		theme       themeState

		windowTitle string
		lastLayer   func(*view.Editor) layerFunc
		images      *imageRegistry
		fileWatcher *fileWatcher

		graphemeClustering bool
	}

	compositionState struct {
		singleLayer bool
		regions     []geom.Area
		precise     bool
		changed     bool
	}

	pickerState struct {
		saved  map[string]*Picker
		layout PickerLayoutOptions
	}

	themeState struct {
		active     *theme.Theme
		dimmed     *theme.Theme
		name       string
		dim        int
		generation int
	}
)

const (
	maxInactiveDim = 90
	fullBrightness = 100
)

// StyleGen returns a counter that increments whenever the active theme changes
func (c *Context) StyleGen() int {
	return c.theme.generation
}

// Theme returns the active theme, reloading it when the configured name changes
func (c *Context) Theme() *theme.Theme {
	c.ensureTheme()
	return c.theme.active
}

// ThemeFor returns the active theme, dimmed for an unfocused pane
func (c *Context) ThemeFor(focused bool) *theme.Theme {
	c.ensureTheme()
	if focused {
		return c.theme.active
	}
	return c.theme.dimmed
}

func (c *Context) ensureTheme() {
	name := c.Editor.Options().Theme
	dim := min(max(c.Editor.Options().InactiveDim, 0), maxInactiveDim)
	if name == c.theme.name && dim == c.theme.dim {
		return
	}
	if name != c.theme.name {
		c.theme.name = name
		c.theme.active = paletteFor(loadTheme(name))
	}
	c.theme.dim = dim
	c.theme.dimmed = paletteFor(c.theme.active.Dimmed(fullBrightness - dim))
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
