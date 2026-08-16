package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type promptHandler func(*view.Editor, string) error

// WithStartupCmd returns a model that runs cmd once on start
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

// WithInitialPicker returns a model that opens a picker on start
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

// PickerAction returns an action opening the picker fn builds
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
		ec.queueNextLayer(layer)
	}
}

// CmdModeAction opens the command prompt
func (m Model) CmdModeAction(_ *view.Editor) {
	ec := m.component
	head := ec.overlayHead()
	ec.queueNextLayer(func(cx *Context) (Component, tea.Cmd) {
		return newPromptComponent(promptComponentArgs{
			cx:     cx,
			editor: ec,
			kind:   promptCmd,
			head:   head,
		}), nil
	})
}

// SearchAction returns an action opening the search prompt
func (m Model) SearchAction(forward bool) command.Action {
	ec := m.component
	return func(*view.Editor) {
		head := ec.overlayHead()
		ec.queueNextLayer(func(cx *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				cx:      cx,
				editor:  ec,
				kind:    promptSearch,
				forward: forward,
				head:    head,
			}), nil
		})
	}
}

// RegexAction returns an action prompting for a pattern, then running fn
func (m Model) RegexAction(prompt i18n.Key, fn promptHandler) command.Action {
	ec := m.component
	return func(*view.Editor) {
		head := ec.overlayHead()
		ec.queueNextLayer(func(cx *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				cx:       cx,
				editor:   ec,
				kind:     promptRegex,
				titleKey: prompt,
				head:     head,
				handler:  fn,
			}), nil
		})
	}
}

// ShellAction returns an action prompting for a command, then running fn
func (m Model) ShellAction(prompt i18n.Key, fn promptHandler) command.Action {
	ec := m.component
	return func(*view.Editor) {
		head := ec.overlayHead()
		ec.queueNextLayer(func(cx *Context) (Component, tea.Cmd) {
			return newPromptComponent(promptComponentArgs{
				cx:       cx,
				editor:   ec,
				kind:     promptShell,
				titleKey: prompt,
				head:     head,
				handler:  fn,
			}), nil
		})
	}
}

// CommandPaletteAction opens the command palette
func (m Model) CommandPaletteAction(e *view.Editor) {
	ec := m.component
	cx := m.context
	opener := func(e *view.Editor) layerFunc {
		p := m.CommandPalettePicker(e)
		cmd := p.load.feedCmd
		p.load.feedCmd = nil
		return func(cx *Context) (Component, tea.Cmd) {
			return newPickerComponent(cx, p), cmd
		}
	}

	cx.lastLayer = opener
	ec.queueNextLayer(opener(e))
}

// LastPickerAction reopens the picker used most recently
func (m Model) LastPickerAction(e *view.Editor) {
	ec := m.component
	cx := m.context
	if cx.lastLayer == nil {
		return
	}
	ec.queueNextLayer(cx.lastLayer(e))
}

// AboutAction opens the about popup
func (m Model) AboutAction(_ *view.Editor) {
	m.component.queueNextLayer(func(*Context) (Component, tea.Cmd) {
		return &aboutComponent{}, nil
	})
}

// MacroRecordAction starts or stops recording a macro
func (m Model) MacroRecordAction(e *view.Editor) command.Continuation {
	return m.component.MacroRecordAction(e)
}

// MacroReplayAction replays the recorded macro
func (m Model) MacroReplayAction(e *view.Editor) command.Continuation {
	return m.component.MacroReplayAction(e)
}
