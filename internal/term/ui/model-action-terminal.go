package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

// TerminalAction opens the user's shell in the focused pane
func (m Model) TerminalAction() command.KeyAction {
	return func(e *view.Editor) command.Continuation {
		tp, err := NewTerminalPane(interactiveShell(), 0, 0)
		if err != nil {
			e.SetStatusMsg(err.Error())
			return nil
		}
		tp.restore = e.ReplacePane(e.Tree().Focus(), tp)
		return nil
	}
}

// TerminalSearchAction opens a prompt that jumps the focused terminal's
// scrollback to the nearest match above the current view
func (m Model) TerminalSearchAction() command.KeyAction {
	ec := m.component
	return func(e *view.Editor) command.Continuation {
		tp, ok := e.Tree().Get(e.Tree().Focus()).(*TerminalPane)
		if !ok {
			return nil
		}
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec: ec, kind: promptTerminalSearch, prompt: "scrollback:",
				fn: func(_ *view.Editor, s string) error {
					if !tp.SearchScrollback(s) {
						return ErrScrollbackNoMatch
					}
					return nil
				},
			}), nil
		}
		return nil
	}
}

// RestoreTerminalPanes reopens a shell in each pane a restored session
// marked as a terminal, since a live shell can't be saved
func (m Model) RestoreTerminalPanes(e *view.Editor) {
	for _, id := range e.TakePendingTerminals() {
		tp, err := NewTerminalPane(interactiveShell(), 0, 0)
		if err != nil {
			continue
		}
		old := e.ReplacePane(id, tp)
		closeTerminalChain(old)
		e.DiscardPane(old)
	}
}
