package ui

import (
	"runtime"
	"runtime/debug"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
)

type aboutComponent struct {
	overlayBuf
	lines []popupLine
}

const (
	AboutVersion     i18n.Key = "about.version"
	AboutDevelopment i18n.Key = "about.development"
)

const (
	// The popup is bounded so it stays readable on a large screen
	aboutMaxWidth  = 52
	aboutMaxHeight = 16

	// Branding and legal notices are not translated
	aboutTitle     = "Thom's Own Editor (toe)"
	aboutTagline   = "A modal text editor for Go development"
	aboutCopyright = "Copyright (c) 2026 Thomas S. Bradford"
	aboutLicense   = "MIT License"
	aboutURL       = "https://github.com/kode4food/toe"
)

var _ BufferOverlayComponent = (*aboutComponent)(nil)

// HandleEvent dismisses the popup on any key or click
func (a *aboutComponent) HandleEvent(
	_ *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyEscape, tea.KeyEnter:
			return consumedWith(popLayer), nil
		}
		return ignoredWith(popLayer), nil
	case tea.MouseClickMsg:
		return consumedWith(popLayer), nil
	case tea.MouseWheelMsg:
		// swallowed so the document behind the popup does not scroll
		return consumed(), nil
	}
	return ignored(), nil
}

// Cursor leaves the cursor to the layer below
func (a *aboutComponent) Cursor(*Context, geom.Size) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

// Layout centers the popup on the screen
func (a *aboutComponent) Layout(
	_ *Context, screen geom.Size,
) (geom.Area, bool) {
	lines, size := measureTextPopup(geom.Size{
		Width:  min(screen.Width, aboutMaxWidth),
		Height: min(screen.Height, aboutMaxHeight),
	}, aboutText())
	a.lines = lines
	return geom.Area{
		Point: geom.Point{
			X: max((screen.Width-size.Width)/2, 0),
			Y: max((screen.Height-size.Height)/2, 0),
		},
		Size: size,
	}, true
}

// PaintBuffer draws the about popup
func (a *aboutComponent) PaintBuffer(cx *Context, pl geom.Area) *tui.Buffer {
	return a.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		paintTextPopup(cx, buf, a.lines)
	})
}

func aboutText() string {
	version := i18n.Text(AboutDevelopment)
	if v := buildVersion(); v != "" {
		version = i18n.Text(AboutVersion) + " " + v
	}
	platform := runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH
	return strings.Join([]string{
		"# " + aboutTitle,
		"",
		aboutTagline,
		"",
		version,
		platform,
		"",
		aboutCopyright,
		aboutLicense,
		"",
		aboutURL,
	}, "\n")
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" || strings.Contains(v, "+") ||
		strings.Count(v, "-") > 1 {
		return ""
	}
	return v
}
