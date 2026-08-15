package ui

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

// ResizeViewAction starts interactive split resizing
func (m Model) ResizeViewAction(e *view.Editor) command.Continuation {
	e.Tree().Unmaximize()
	var cont command.Continuation
	cont = command.PopOnBackspace(func(
		e *view.Editor, k command.KeyEvent,
	) (command.Continuation, command.Transition) {
		if k.Mods != command.ModNone {
			return cont, command.ContinuationStay
		}
		switch {
		case k.Code.Special == command.Escape, k.Code.Special == command.Enter:
			return nil, command.ContinuationDone
		case k.Code.Char == 'h', k.Code.Special == command.Left:
			action.ResizeViewLeft(e)
		case k.Code.Char == 'l', k.Code.Special == command.Right:
			action.ResizeViewRight(e)
		case k.Code.Char == 'j', k.Code.Special == command.Down:
			action.ResizeViewDown(e)
		case k.Code.Char == 'k', k.Code.Special == command.Up:
			action.ResizeViewUp(e)
		}
		return cont, command.ContinuationStay
	})
	return cont
}
