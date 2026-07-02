// Package ui implements the Bubbletea terminal application model for toe
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
)

// Model is the root Bubbletea model — a thin wrapper around Compositor
type Model struct {
	compositor *Compositor
	context    *Context
	component  *EditorComponent
	initCmd    tea.Cmd
}

// New creates an initialized Model for the given editor and keymaps
func New(editor *view.Editor, km *command.Keymaps) Model {
	cx := &Context{Editor: editor, Keymaps: km, Syntax: syntax.NewSyntaxCache()}
	ec := newEditorComponent()
	comp := &Compositor{}
	comp.Push(ec)
	return Model{
		compositor: comp,
		context:    cx,
		component:  ec,
		initCmd:    ec.fileWatchCmd(cx),
	}
}

// CompletionOptions returns the UI-owned completion popup behavior settings
func (m Model) CompletionOptions() CompletionOptions {
	return m.component.completionOpts
}

// SetCompletionOptions applies UI-owned completion popup behavior settings
func (m Model) SetCompletionOptions(opts CompletionOptions) {
	m.component.completionOpts = opts.WithDefaults()
}

// Init fires the startup cmd if one was set before the program started
func (m Model) Init() tea.Cmd {
	return m.initCmd
}

// Update delegates all events to the compositor
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.compositor.HandleEvent(msg, m.context)
	m.component.syncFileWatcher(m.context)
	return m, cmd
}

// View renders the current frame via the compositor
func (m Model) View() tea.View {
	if m.compositor.width == 0 || m.compositor.height == 0 {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	v := tea.NewView(m.compositor.Render(m.context))
	v.AltScreen = true
	if m.context.Editor.Options().Mouse {
		v.MouseMode = tea.MouseModeCellMotion
	}
	v.ReportFocus = true
	if cur, ok := m.compositor.Cursor(m.context); ok {
		v.Cursor = &cur
	}
	return v
}
