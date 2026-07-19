// Package ui implements the Bubbletea terminal application model for toe
package ui

import (
	"time"

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
	editor.SetIndenter(func(doc *view.Document, line, pos int) (string, bool) {
		return syntax.IndentForNewline(
			doc.Text(), doc.Lang(), line, pos, doc.IndentStyle(),
		)
	})
	registerImagePane(editor)
	registerTerminalPane(editor)
	cx := &Context{
		Editor:       editor,
		Keymaps:      km,
		Syntax:       syntax.NewSyntaxCache(),
		images:       newImageRegistry(),
		pickerLayout: PickerLayoutOptions{},
	}
	ec := newEditorComponent()
	comp := &Compositor{}
	comp.Push(ec)
	return Model{
		compositor: comp,
		context:    cx,
		component:  ec,
		initCmd: tea.Batch(
			ec.fileWatchCmd(cx),
			vcsUpdateCmd(cx),
			vcsRefreshCmd(cx),
			spinnerTickCmd(),
			terminalPollCmd(),
		),
	}
}

// PickerLayoutOptions returns the UI-owned picker layout settings
func (m Model) PickerLayoutOptions() PickerLayoutOptions {
	return m.context.pickerLayout.clone()
}

// SetPickerLayoutOptions applies UI-owned picker layout settings
func (m Model) SetPickerLayoutOptions(opts PickerLayoutOptions) {
	opts = opts.clone()
	for key, ratio := range opts.SplitRatios {
		opts.SplitRatios[key] = clampPickerSplitRatio(ratio)
	}
	m.context.pickerLayout = opts
}

// Init fires the startup cmd if one was set before the program started
func (m Model) Init() tea.Cmd {
	return m.initCmd
}

// Update delegates all events to the compositor
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imageDisplayRequestMsg:
		if msg.gen != m.context.imageGen {
			return m, nil
		}
		return m, tea.Batch(m.imageDisplayCmd(), m.pickerImageCmd())
	case imageTransmitMsg:
		// The encode finished off the event loop; write the escape stream and
		// then mark the image ready so its placeholder cells emit
		return m, tea.Sequence(tea.Raw(msg.raw), func() tea.Msg {
			return imageReadyMsg{id: msg.id, size: msg.size}
		})
	case imageReadyMsg:
		if m.context.images.placed[msg.id] != msg.size {
			return m, nil
		}
		// the transmit has now been written; mark the image surfaces dirty so
		// they repaint and, with ready set, actually emit their placeholder
		// cells for the terminal to composite the picture over
		m.context.images.ready[msg.id] = msg.size
		m.markImageDirty()
		return m, nil
	default:
		cmd := m.compositor.HandleEvent(msg, m.context)
		m.component.syncFileWatcher(m.context)
		return m, tea.Batch(cmd, m.scheduleImageDisplayCmd())
	}
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

func (m Model) pickerImageCmd() tea.Cmd {
	if p, ok := m.compositor.activePicker(); ok {
		return p.previewImageCmd(
			m.context, m.compositor.width, m.compositor.height,
		)
	}
	return nil
}

func (m Model) scheduleImageDisplayCmd() tea.Cmd {
	if !m.hasImageSurface() {
		return nil
	}
	m.context.imageGen++
	gen := m.context.imageGen
	return func() tea.Msg {
		time.Sleep(imageDisplayDelay)
		return imageDisplayRequestMsg{gen: gen}
	}
}

func (m Model) hasImageSurface() bool {
	if p, ok := m.compositor.activePicker(); ok {
		if p.hasPreviewImage(
			m.context, m.compositor.width, m.compositor.height,
		) {
			return true
		}
	}
	found := false
	m.context.Editor.Tree().Range(func(p view.Pane) bool {
		_, found = p.(*ImagePane)
		return !found
	})
	return found
}

func (m Model) markImageDirty() {
	m.context.Editor.Tree().Range(func(p view.Pane) bool {
		if _, ok := p.(*ImagePane); ok {
			p.MarkDirty()
		}
		return true
	})
	if p, ok := m.compositor.activePicker(); ok {
		p.markDirty()
	}
}
