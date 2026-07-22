package ui

import (
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

// ResizeViewAction enters an interactive resize mode: h/l (or left/right) and
// j/k (or up/down) push the focused split's border in that literal screen
// direction, one cell per keypress, until Escape or Enter exits
func (m Model) ResizeViewAction(e *view.Editor) command.Continuation {
	e.SetHint(i18n.Text(i18n.HintResize))
	var cont command.Continuation
	cont = func(e *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Mods != command.ModNone {
			return cont
		}
		switch {
		case k.Code.Special == command.Escape, k.Code.Special == command.Enter:
			return nil
		case k.Code.Char == 'h', k.Code.Special == command.Left:
			action.ResizeViewLeft(e)
		case k.Code.Char == 'l', k.Code.Special == command.Right:
			action.ResizeViewRight(e)
		case k.Code.Char == 'j', k.Code.Special == command.Down:
			action.ResizeViewDown(e)
		case k.Code.Char == 'k', k.Code.Special == command.Up:
			action.ResizeViewUp(e)
		}
		e.SetHint(i18n.Text(i18n.HintResize))
		return cont
	}
	return cont
}
