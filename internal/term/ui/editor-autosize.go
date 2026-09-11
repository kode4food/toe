package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
)

type (
	autoSizeState struct {
		enabled     bool
		viewID      view.Id
		area        geom.Area
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

func (ec *EditorComponent) autoSizeCmd() tea.Cmd {
	cx := ec.context
	id := cx.Editor.Tree().Focus()
	pane := cx.Editor.FocusedPane()
	if pane == nil {
		return nil
	}
	area := pane.Area()
	if id == ec.autoSize.viewID && area == ec.autoSize.area {
		return nil
	}
	ec.cancelAutoSize()
	ec.autoSize.viewID = id
	ec.autoSize.area = area
	if !ec.autoSize.enabled {
		return nil
	}
	ec.autoSize.targetWidth = ec.autoSizeWidthTarget(pane)
	if ec.autoSize.targetWidth == 0 {
		return nil
	}
	if !ec.animation {
		grow := ec.autoSize.targetWidth - pane.Area().Width
		cx.Editor.Tree().GrowFocusedWidth(grow)
		ec.autoSize.area = pane.Area()
		return nil
	}
	ec.autoSize.generation++
	return tea.Batch(
		autoSizeTickCmd(ec.autoSize.generation), ec.settlePaneResizeCmd(),
	)
}

func (ec *EditorComponent) autoSizeWidthTarget(pane view.Pane) int {
	var want int
	switch p := pane.(type) {
	case *view.View:
		want = ec.autoSizeRulerWidth(p)
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

func (ec *EditorComponent) autoSizeRulerWidth(v *view.View) int {
	cx := ec.context
	rulers := cx.Editor.Options().Rulers
	doc := cx.Editor.Document(v.DocID())
	if len(rulers) == 0 || rulers[0] <= 0 || doc == nil {
		return 0
	}
	return gutterWidthFor(doc.Text(), cx.Editor.Options().Gutters) +
		rulers[0] + 1
}

func (ec *EditorComponent) cancelAutoSize() {
	ec.autoSize.targetWidth = 0
	ec.autoSize.generation++
}

func (ec *EditorComponent) cancelAutoSizeFor(msg tea.Msg) {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseClickMsg, tea.WindowSizeMsg:
		ec.cancelAutoSize()
	}
}

func (ec *EditorComponent) handleAutoSizeTick(
	msg autoSizeTickMsg,
) (EventResult, tea.Cmd) {
	if msg.generation != ec.autoSize.generation ||
		ec.autoSize.targetWidth == 0 {
		return consumed(), nil
	}
	cx := ec.context
	pane := cx.Editor.FocusedPane()
	if pane == nil || pane.ID() != ec.autoSize.viewID {
		ec.cancelAutoSize()
		return consumed(), nil
	}
	before := pane.Area().Width
	step := min(autoSizeStep, ec.autoSize.targetWidth-before)
	grew := cx.Editor.Tree().GrowFocusedWidth(step)
	ec.autoSize.area = pane.Area()
	if !grew || pane.Area().Width <= before ||
		pane.Area().Width >= ec.autoSize.targetWidth {
		ec.cancelAutoSize()
		return consumed(), nil
	}
	return consumed(), tea.Batch(
		autoSizeTickCmd(msg.generation), ec.settlePaneResizeCmd(),
	)
}

func autoSizeTickCmd(generation int) tea.Cmd {
	return tea.Tick(autoSizeTickInterval, func(time.Time) tea.Msg {
		return autoSizeTickMsg{generation: generation}
	})
}
