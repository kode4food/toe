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

func (ec *EditorComponent) settlePaneResizeCmd(cx *Context) tea.Cmd {
	if len(ec.resizeHold.held) == 0 {
		cx.Editor.Tree().Range(func(p view.Pane) bool {
			if holder, ok := p.(view.ResizeHolder); ok {
				holder.HoldResize()
				ec.resizeHold.held = append(ec.resizeHold.held, holder)
			}
			return true
		})
	}
	ec.resizeHold.generation++
	generation := ec.resizeHold.generation
	return tea.Tick(resizeSettleDelay, func(time.Time) tea.Msg {
		return resizeSettleMsg{generation: generation}
	})
}

func (ec *EditorComponent) handleResizeSettle(
	msg resizeSettleMsg,
) (EventResult, tea.Cmd) {
	if msg.generation != ec.resizeHold.generation {
		return consumed(), nil
	}
	for _, holder := range ec.resizeHold.held {
		holder.ResumeResize()
	}
	ec.resizeHold.held = nil
	return consumed(), nil
}
