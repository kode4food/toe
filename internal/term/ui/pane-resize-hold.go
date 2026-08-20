package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/view"
)

type (
	resizeHoldState struct {
		held       []view.ResizeHolder
		generation int
	}

	resizeSettleMsg struct{ generation int }
)

const resizeSettleDelay = 120 * time.Millisecond

func (e *EditorComponent) settlePaneResizeCmd(cx *Context) tea.Cmd {
	if len(e.resizeHold.held) == 0 {
		cx.Editor.Tree().Range(func(p view.Pane) bool {
			if holder, ok := p.(view.ResizeHolder); ok {
				holder.HoldResize()
				e.resizeHold.held = append(e.resizeHold.held, holder)
			}
			return true
		})
	}
	e.resizeHold.generation++
	generation := e.resizeHold.generation
	return tea.Tick(resizeSettleDelay, func(time.Time) tea.Msg {
		return resizeSettleMsg{generation: generation}
	})
}

func (e *EditorComponent) handleResizeSettle(
	msg resizeSettleMsg,
) (EventResult, tea.Cmd) {
	if msg.generation != e.resizeHold.generation {
		return consumed(), nil
	}
	for _, holder := range e.resizeHold.held {
		holder.ResumeResize()
	}
	e.resizeHold.held = nil
	return consumed(), nil
}
