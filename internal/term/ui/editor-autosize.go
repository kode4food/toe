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
	autoSizeTermWidth    = 80
)

// AutoSize reports whether focused panes widen to fit their content
func (m Model) AutoSize() bool {
	return m.component.autoSize.enabled
}

// SetAutoSize controls whether focused panes widen to fit their content
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
	pane := cx.Editor.FocusedPane()
	if pane == nil {
		return nil
	}
	e.autoSize.targetWidth = e.autoSizeWidthTarget(cx, pane)
	if e.autoSize.targetWidth == 0 {
		return nil
	}
	if !e.animation {
		grow := e.autoSize.targetWidth - pane.Area().Width
		cx.Editor.Tree().GrowFocusedWidth(grow)
		return nil
	}
	e.autoSize.generation++
	return tea.Batch(
		autoSizeTickCmd(e.autoSize.generation), e.settlePaneResizeCmd(cx),
	)
}

// image panes scale to whatever width they are given, so no case grows them
func (e *EditorComponent) autoSizeWidthTarget(
	cx *Context, pane view.Pane,
) int {
	var want int
	switch p := pane.(type) {
	case *view.View:
		want = e.autoSizeRulerWidth(cx, p)
	case *BinaryPane:
		want = binaryTargetWidth(p.offsetWidth())
	case *TerminalPane:
		want = autoSizeTermWidth
	}
	if want <= pane.Area().Width {
		return 0
	}
	return want
}

func (e *EditorComponent) autoSizeRulerWidth(
	cx *Context, v *view.View,
) int {
	rulers := cx.Editor.Options().Rulers
	doc := cx.Editor.Document(v.DocID())
	if len(rulers) == 0 || rulers[0] <= 0 || doc == nil {
		return 0
	}
	return gutterWidthFor(doc.Text(), cx.Editor.Options().Gutters) +
		rulers[0] + 1
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
	pane := cx.Editor.FocusedPane()
	if pane == nil || pane.ID() != e.autoSize.viewID {
		e.cancelAutoSize()
		return consumed(), nil
	}
	before := pane.Area().Width
	step := min(autoSizeStep, e.autoSize.targetWidth-before)
	grew := cx.Editor.Tree().GrowFocusedWidth(step)
	if !grew || pane.Area().Width <= before ||
		pane.Area().Width >= e.autoSize.targetWidth {
		e.cancelAutoSize()
		return consumed(), nil
	}
	return consumed(), tea.Batch(
		autoSizeTickCmd(msg.generation), e.settlePaneResizeCmd(cx),
	)
}

func autoSizeTickCmd(generation int) tea.Cmd {
	return tea.Tick(autoSizeTickInterval, func(time.Time) tea.Msg {
		return autoSizeTickMsg{generation: generation}
	})
}
