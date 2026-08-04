package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/view"
)

type (
	autoSizeState struct {
		enabled      bool
		verticalPct  int
		viewID       view.Id
		startWidth   int
		startHeight  int
		targetWidth  int
		targetHeight int
		frame        int
		frames       int
		generation   int
	}

	autoSizeTickMsg struct{ generation int }
)

// DefaultAutoSizeVerticalPct is the percent of the parent split a pane grows
// to vertically when no percent is configured
const DefaultAutoSizeVerticalPct = 67

const (
	autoSizeTickInterval = 12 * time.Millisecond
	autoSizeStep         = 3
	autoSizeCellAspect   = 2
)

// AutoSize reports whether focused editor panes grow to show the leftmost ruler
func (m Model) AutoSize() bool {
	return m.component.autoSize.enabled
}

// SetAutoSize controls whether focused editor panes grow to show their leftmost
// ruler
func (m Model) SetAutoSize(enabled bool) {
	m.component.cancelAutoSize()
	m.component.autoSize.enabled = enabled
	m.component.autoSize.viewID = view.InvalidViewId
}

// AutoSizeVerticalPercent reports the percent of its parent split a focused
// pane grows to vertically, or 0 when vertical auto-sizing is off
func (m Model) AutoSizeVerticalPercent() int {
	return m.component.autoSize.verticalPct
}

// SetAutoSizeVerticalPercent sets the vertical growth percent, clamped to
// [0, 100]; 0 disables vertical auto-sizing
func (m Model) SetAutoSizeVerticalPercent(percent int) {
	m.component.cancelAutoSize()
	m.component.autoSize.verticalPct = max(0, min(100, percent))
	m.component.autoSize.viewID = view.InvalidViewId
}

func (e *EditorComponent) autoSizeCmd(cx *Context) tea.Cmd {
	id := cx.Editor.Tree().Focus()
	if id == e.autoSize.viewID {
		return nil
	}
	e.cancelAutoSize()
	e.autoSize.viewID = id
	if !e.autoSize.enabled {
		return nil
	}
	v := cx.Editor.FocusedView()
	doc := cx.Editor.FocusedDocument()
	if v == nil || doc == nil {
		return nil
	}
	e.autoSize.startWidth = v.Area().Width
	e.autoSize.startHeight = v.Area().Height
	e.autoSize.targetWidth = e.autoSizeWidthTarget(cx, v, doc)
	e.autoSize.targetHeight = e.autoSizeHeightTarget(cx, v)
	var dw, dh int
	if e.autoSize.targetWidth != 0 {
		dw = e.autoSize.targetWidth - e.autoSize.startWidth
	}
	if e.autoSize.targetHeight != 0 {
		dh = e.autoSize.targetHeight - e.autoSize.startHeight
	}
	if dw == 0 && dh == 0 {
		return nil
	}
	if !e.animation {
		tree := cx.Editor.Tree()
		if dw > 0 {
			tree.GrowFocusedWidth(dw)
		}
		if dh > 0 {
			tree.GrowFocusedHeight(dh)
		}
		return nil
	}
	// pace both axes over one shared frame count, so linear interpolation lands
	// them together; rows weighted by cell aspect keep the visual speed even
	visual := max(dw, dh*autoSizeCellAspect)
	e.autoSize.frame = 0
	e.autoSize.frames = max((visual+autoSizeStep-1)/autoSizeStep, 1)
	e.autoSize.generation++
	return autoSizeTickCmd(e.autoSize.generation)
}

// autoSizeWidthTarget is the pane width that reveals the leftmost ruler, or 0
// if the pane is already wide enough
func (e *EditorComponent) autoSizeWidthTarget(
	cx *Context, v *view.View, doc *view.Document,
) int {
	rulers := cx.Editor.Options().Rulers
	if len(rulers) == 0 || rulers[0] <= 0 {
		return 0
	}
	want := gutterWidthFor(doc.Text(), cx.Editor.Options().Gutters) +
		rulers[0] + 1
	if want <= v.Area().Width {
		return 0
	}
	return want
}

// autoSizeHeightTarget is verticalPercent of the parent split height, or 0 when
// off, already tall enough, or there is no vertical split
func (e *EditorComponent) autoSizeHeightTarget(cx *Context, v *view.View) int {
	if e.autoSize.verticalPct <= 0 {
		return 0
	}
	parentH, ok := cx.Editor.Tree().FocusedParentHeight()
	if !ok {
		return 0
	}
	want := parentH * e.autoSize.verticalPct / 100
	if want <= v.Area().Height {
		return 0
	}
	return want
}

func (e *EditorComponent) cancelAutoSize() {
	e.autoSize.targetWidth = 0
	e.autoSize.targetHeight = 0
	e.autoSize.generation++
}

func (e *EditorComponent) cancelAutoSizeFor(msg tea.Msg) {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseClickMsg, tea.WindowSizeMsg:
		e.cancelAutoSize()
	}
}

func (e *EditorComponent) handleAutoSizeTick(
	cx *Context, msg autoSizeTickMsg,
) (EventResult, tea.Cmd) {
	if msg.generation != e.autoSize.generation ||
		(e.autoSize.targetWidth == 0 && e.autoSize.targetHeight == 0) {
		return consumed(), nil
	}
	v := cx.Editor.FocusedView()
	if v == nil || v.ID() != e.autoSize.viewID {
		e.cancelAutoSize()
		return consumed(), nil
	}
	e.autoSize.frame++
	frame := e.autoSize.frame
	frames := e.autoSize.frames
	wantW := e.autoSize.startWidth +
		(e.autoSize.targetWidth-e.autoSize.startWidth)*frame/frames
	wantH := e.autoSize.startHeight +
		(e.autoSize.targetHeight-e.autoSize.startHeight)*frame/frames
	tree := cx.Editor.Tree()
	before := v.Area()
	if delta := wantW - before.Width; e.autoSize.targetWidth != 0 && delta > 0 {
		if !tree.GrowFocusedWidth(delta) || v.Area().Width <= before.Width {
			e.autoSize.targetWidth = 0
		}
	}
	if delta := wantH - before.Height; e.autoSize.targetHeight != 0 && delta > 0 {
		if !tree.GrowFocusedHeight(delta) || v.Area().Height <= before.Height {
			e.autoSize.targetHeight = 0
		}
	}
	if v.Area().Width >= e.autoSize.targetWidth {
		e.autoSize.targetWidth = 0
	}
	if v.Area().Height >= e.autoSize.targetHeight {
		e.autoSize.targetHeight = 0
	}
	if e.autoSize.targetWidth == 0 && e.autoSize.targetHeight == 0 {
		return consumed(), nil
	}
	return consumed(), autoSizeTickCmd(msg.generation)
}

func autoSizeTickCmd(generation int) tea.Cmd {
	return tea.Tick(autoSizeTickInterval, func(time.Time) tea.Msg {
		return autoSizeTickMsg{generation: generation}
	})
}
