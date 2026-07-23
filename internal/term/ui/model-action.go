package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type promptHandler func(*view.Editor, string) error

func (m Model) WithStartupCmd(cmd tea.Cmd) Model {
	m.initCmd = tea.Batch(m.initCmd, cmd)
	return m
}

// WithStartupMessage sets a status bar message for the first frame
func (m Model) WithStartupMessage(msg string) Model {
	if msg == "" {
		m.component.clearCommandMessage()
		return m
	}
	m.component.setCommandMessage(msg)
	return m
}

func (m Model) WithInitialPicker(fn PickerFunc) Model {
	m.compositor.startup = func(cx *Context) (Component, tea.Cmd) {
		p := fn(cx.Editor)
		if p == nil {
			return nil, nil
		}
		cmd := p.feedCmd
		p.feedCmd = nil
		return newPickerComponent(p), cmd
	}
	return m
}

func (m Model) PickerAction(fn PickerFunc) command.KeyAction {
	ec := m.component
	cx := m.context
	opener := func(e *view.Editor) layerFunc {
		p := fn(e)
		if p == nil {
			return nil
		}
		cmd := p.feedCmd
		p.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
		}
	}
	return func(e *view.Editor) command.Continuation {
		layer := opener(e)
		if layer == nil {
			return nil
		}
		cx.lastLayer = opener
		ec.nextLayer = layer
		return nil
	}
}

func (m Model) CmdModeAction() command.KeyAction {
	ec := m.component
	return func(_ *view.Editor) command.Continuation {
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec:   ec,
				kind: promptCmd,
			}), nil
		}
		return nil
	}
}

func (m Model) SearchAction(forward bool) command.KeyAction {
	ec := m.component
	return func(_ *view.Editor) command.Continuation {
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec:      ec,
				kind:    promptSearch,
				forward: forward,
			}), nil
		}
		return nil
	}
}

func (m Model) RegexAction(prompt string, fn promptHandler) command.KeyAction {
	ec := m.component
	return func(_ *view.Editor) command.Continuation {
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec:     ec,
				kind:   promptRegex,
				prompt: prompt,
				fn:     fn,
			}), nil
		}
		return nil
	}
}

func (m Model) ShellAction(prompt string, fn promptHandler) command.KeyAction {
	ec := m.component
	return func(_ *view.Editor) command.Continuation {
		ec.nextLayer = func(_ *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				ec:     ec,
				kind:   promptShell,
				prompt: prompt,
				fn:     fn,
			}), nil
		}
		return nil
	}
}

func (m Model) CommandPaletteAction() command.KeyAction {
	ec := m.component
	cx := m.context
	opener := func(e *view.Editor) layerFunc {
		p := CommandPalettePicker(e, cx.Keymaps)
		cmd := p.feedCmd
		p.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
		}
	}
	return func(e *view.Editor) command.Continuation {
		cx.lastLayer = opener
		ec.nextLayer = opener(e)
		return nil
	}
}

func (m Model) LastPickerAction() command.KeyAction {
	ec := m.component
	cx := m.context
	return func(e *view.Editor) command.Continuation {
		if cx.lastLayer == nil {
			return nil
		}
		ec.nextLayer = cx.lastLayer(e)
		return nil
	}
}

func (m Model) MacroRecordAction(e *view.Editor) command.Continuation {
	return m.component.MacroRecordAction(e)
}

func (m Model) MacroReplayAction(e *view.Editor) command.Continuation {
	return m.component.MacroReplayAction(e)
}
