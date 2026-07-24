package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

// TerminalAction opens the user's shell in the focused pane
func (m Model) TerminalAction() command.KeyAction {
	return func(e *view.Editor) command.Continuation {
		if _, ok := e.Tree().Get(e.Tree().Focus()).(*TerminalPane); ok {
			return nil
		}
		tp, err := NewTerminalPane(e, interactiveShell(), geom.Size{})
		if err != nil {
			e.SetStatusMsg(i18n.ErrorText(err))
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
		ec.keys.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec:     ec,
				kind:   promptTerminalSearch,
				prompt: i18n.Text(i18n.PromptScrollbackSearch),
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
