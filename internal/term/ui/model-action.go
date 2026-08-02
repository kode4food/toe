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
		cmd := p.load.feedCmd
		p.load.feedCmd = nil
		return newPickerComponent(cx, p), cmd
	}
	return m
}

func (m Model) PickerAction(fn PickerFunc) command.Action {
	ec := m.component
	cx := m.context
	opener := func(e *view.Editor) layerFunc {
		p := fn(e)
		if p == nil {
			return nil
		}
		cmd := p.load.feedCmd
		p.load.feedCmd = nil
		return func(cx *Context) (Component, tea.Cmd) {
			return newPickerComponent(cx, p), cmd
		}
	}
	return func(e *view.Editor) {
		layer := opener(e)
		if layer == nil {
			return
		}
		cx.lastLayer = opener
		ec.keys.nextLayer = layer
	}
}

func (m Model) CmdModeAction(_ *view.Editor) {
	ec := m.component
	ec.keys.nextLayer = func(cx *Context) (Component, tea.Cmd) {
		return newPromptComponent(cx, promptComponentArgs{
			ec:   ec,
			kind: promptCmd,
		}), nil
	}
}

func (m Model) SearchAction(forward bool) command.Action {
	ec := m.component
	return func(_ *view.Editor) {
		ec.keys.nextLayer = func(cx *Context) (Component, tea.Cmd) {
			return newPromptComponent(cx, promptComponentArgs{
				ec:      ec,
				kind:    promptSearch,
				forward: forward,
			}), nil
		}
	}
}

func (m Model) RegexAction(prompt string, fn promptHandler) command.Action {
	ec := m.component
	return func(_ *view.Editor) {
		ec.keys.nextLayer = func(cx *Context) (Component, tea.Cmd) {
			return newPromptComponent(cx, promptComponentArgs{
				ec:     ec,
				kind:   promptRegex,
				prompt: prompt,
				fn:     fn,
			}), nil
		}
	}
}

func (m Model) ShellAction(prompt string, fn promptHandler) command.Action {
	ec := m.component
	return func(_ *view.Editor) {
		ec.keys.nextLayer = func(cx *Context) (Component, tea.Cmd) {
			return newPromptComponent(cx, promptComponentArgs{
				ec:     ec,
				kind:   promptShell,
				prompt: prompt,
				fn:     fn,
			}), nil
		}
	}
}

func (m Model) CommandPaletteAction(e *view.Editor) {
	ec := m.component
	cx := m.context
	opener := func(e *view.Editor) layerFunc {
		p := CommandPalettePicker(e, cx.Keymaps)
		cmd := p.load.feedCmd
		p.load.feedCmd = nil
		return func(cx *Context) (Component, tea.Cmd) {
			return newPickerComponent(cx, p), cmd
		}
	}

	cx.lastLayer = opener
	ec.keys.nextLayer = opener(e)
}

func (m Model) LastPickerAction(e *view.Editor) {
	ec := m.component
	cx := m.context
	if cx.lastLayer == nil {
		return
	}
	ec.keys.nextLayer = cx.lastLayer(e)
}

func (m Model) MacroRecordAction(e *view.Editor) command.Continuation {
	return m.component.MacroRecordAction(e)
}

func (m Model) MacroReplayAction(e *view.Editor) command.Continuation {
	return m.component.MacroReplayAction(e)
}
