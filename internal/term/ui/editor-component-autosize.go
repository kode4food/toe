package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/view"
)

type (
	autoSizeState struct {
		enabled     bool
		viewID      view.Id
		targetWidth int
		generation  int
	}

	autoSizeTickMsg struct{ generation int }
)

const (
	autoSizeTickInterval = 12 * time.Millisecond
	autoSizeStep         = 3
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
	rulers := cx.Editor.Options().Rulers
	if v == nil || doc == nil {
		return nil
	}
	if len(rulers) == 0 || rulers[0] <= 0 {
		return nil
	}
	want := gutterWidthFor(doc.Text(), cx.Editor.Options().Gutters) +
		rulers[0] + 1
	if want <= v.Area().Width {
		return nil
	}
	e.autoSize.targetWidth = want
	e.autoSize.generation++
	return autoSizeTickCmd(e.autoSize.generation)
}

func (e *EditorComponent) cancelAutoSize() {
	e.autoSize.targetWidth = 0
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
		e.autoSize.targetWidth == 0 {
		return consumed(), nil
	}
	v := cx.Editor.FocusedView()
	if v == nil || v.ID() != e.autoSize.viewID {
		e.cancelAutoSize()
		return consumed(), nil
	}
	before := v.Area().Width
	delta := min(e.autoSize.targetWidth-before, autoSizeStep)
	if delta <= 0 || !cx.Editor.Tree().GrowFocusedWidth(delta) {
		e.autoSize.targetWidth = 0
		return consumed(), nil
	}
	if v.Area().Width <= before || v.Area().Width >= e.autoSize.targetWidth {
		e.autoSize.targetWidth = 0
		return consumed(), nil
	}
	return consumed(), autoSizeTickCmd(msg.generation)
}

func autoSizeTickCmd(generation int) tea.Cmd {
	return tea.Tick(autoSizeTickInterval, func(time.Time) tea.Msg {
		return autoSizeTickMsg{generation: generation}
	})
}
