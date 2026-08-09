package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

// TerminalAction opens the user's shell in the focused pane
func (m Model) TerminalAction(e *view.Editor) {
	if _, ok := e.Tree().Get(e.Tree().Focus()).(view.Displaceable); !ok {
		return
	}
	tp, err := NewTerminalPane(e, interactiveShell(), geom.Size{})
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	e.DisplacePane(e.Tree().Focus(), tp)
}

// TerminalSearchAction opens a prompt that jumps the focused terminal's
// scrollback to the nearest match above the current view
func (m Model) TerminalSearchAction(e *view.Editor) {
	ec := m.component
	tp, ok := e.Tree().Get(e.Tree().Focus()).(*TerminalPane)
	if !ok {
		return
	}
	ec.queueNextLayer(func(cx *Context) (Component, tea.Cmd) {
		return newPromptComponent(cx, promptComponentArgs{
			editor: ec,
			kind:   promptTerminalSearch,
			prompt: i18n.Text(i18n.PromptScrollbackSearch),
			handler: func(_ *view.Editor, s string) error {
				if !tp.SearchScrollback(s) {
					return ErrScrollbackNoMatch
				}
				return nil
			},
		}), nil
	})
}
