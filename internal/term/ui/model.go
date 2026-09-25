// Package ui implements the Bubbletea terminal application model for toe
package ui

import (
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Model is the root Bubbletea model, a thin wrapper around Compositor
	Model struct {
		compositor *Compositor
		context    *Context
		component  *EditorComponent
		initCmd    tea.Cmd
	}

	settleTimeoutMsg struct{}
)

// answered with CSI 6 ; height ; width t
const requestCellSizeOp = 16

// New creates an initialized Model for the given editor and keymaps
func New(e *view.Editor, km *command.Keymaps) Model {
	e.SetIndenter(func(doc *view.Document, line, pos int) (string, bool) {
		return syntax.IndentForNewline(syntax.IndentForNewlineArgs{
			Text:  doc.Text(),
			Lang:  doc.Lang(),
			Line:  line,
			Pos:   pos,
			Style: doc.IndentStyle(),
		})
	})
	w := newFileWatcher()
	cx := &Context{
		Editor:       e,
		Keymaps:      km,
		Syntax:       syntax.NewSyntaxCache(),
		images:       newImageRegistry(),
		pickerLayout: PickerLayoutOptions{},
		fileWatcher:  w,
		windowTitle:  workspaceTitle(e.Cwd()),
	}
	ec := newEditorComponent(cx)
	e.Tree().SetRedraw(ec.requestRedraw)
	ec.requestRedraw()
	registerImagePane(e)
	registerTerminalPane(e)
	registerBinaryPane(e)
	comp := &Compositor{}
	comp.Push(ec)
	return Model{
		compositor: comp,
		context:    cx,
		component:  ec,
		initCmd: tea.Batch(
			w.nextCmd(e),
			vcsUpdateCmd(cx),
			ec.redrawCmd(),
			tea.Tick(settleWait, func(time.Time) tea.Msg {
				return settleTimeoutMsg{}
			}),
		),
	}
}

// Close releases the model's long-lived resources, such as file watches
func (m Model) Close() {
	m.context.fileWatcher.close()
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
	for key, scale := range opts.Scales {
		opts.Scales[key] = clampOverlayScale(scale)
	}
	m.context.pickerLayout = opts
}

// CompletionOptions returns the UI-owned automatic completion settings
func (m Model) CompletionOptions() CompletionOptions {
	return m.component.completion
}

// SetCompletionOptions applies UI-owned automatic completion settings
func (m Model) SetCompletionOptions(opts CompletionOptions) {
	m.component.completion = opts
}

// Init fires the startup cmd if one was set before the program started
func (m Model) Init() tea.Cmd {
	return m.initCmd
}

// Update delegates all events to the compositor
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cx := m.context
	switch msg := msg.(type) {
	case imageTransmitMsg:
		// Mark ready only after the escape reaches Bubble Tea's writer
		return m, tea.Sequence(tea.Raw(msg.raw), func() tea.Msg {
			return imageReadyMsg{id: msg.id, state: msg.state}
		})
	case uv.CellSizeEvent:
		if cx.images.setCell(geom.Size{Width: msg.Width, Height: msg.Height}) {
			m.markImageDirty()
		}
		return m, m.imageDisplayFrameCmd()
	case imageReadyMsg:
		cx.images.sent[msg.id] = true
		if cx.images.placed[msg.id] == msg.state {
			// only taking up the placement rearranges cells
			placing := cx.images.ready[msg.id].cells != msg.state.cells
			cx.images.ready[msg.id] = msg.state
			if placing {
				m.markImageDirty()
			}
		}
		// re-query even when a size was requested while this was in flight, so
		// a starved request is retried now that the id is confirmed sent
		return m, m.imageDisplayFrameCmd()
	default:
		m.component.cancelAutoSizeFor(msg)
		cmd := m.compositor.HandleEvent(cx, msg)
		if next := m.component.takeNextLayer(); next != nil {
			layer, nextCmd := next(cx)
			if layer != nil {
				m.compositor.Push(layer)
			}
			cmd = tea.Batch(cmd, nextCmd)
		}
		cx.fileWatcher.sync(cx.Editor)
		return m, tea.Batch(
			cmd, m.component.autoSizeCmd(), m.imageDisplayFrameCmd(),
			cellSizeQueryCmd(msg),
		)
	}
}

// View renders the current frame via the compositor
func (m Model) View() tea.View {
	cx := m.context
	if m.compositor.size.IsEmpty() {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	v := tea.NewView(m.compositor.Render(cx))
	v.AltScreen = true
	v.WindowTitle = cx.windowTitle
	if cx.Editor.Options().Mouse {
		v.MouseMode = tea.MouseModeCellMotion
	}
	v.ReportFocus = true
	if cur, ok := m.compositor.Cursor(cx); ok {
		v.Cursor = &cur
	}
	return v
}

func (m Model) pickerImageCmd() tea.Cmd {
	if p, ok := m.compositor.activePreviewImager(); ok {
		return p.previewImageCmd(m.context, m.compositor.size)
	}
	return nil
}

func (m Model) imageDisplayFrameCmd() tea.Cmd {
	if !m.hasImageSurface() {
		return nil
	}
	m.context.images.beginFrame()
	return tea.Batch(
		m.imageDisplayCmd(), m.pickerImageCmd(), m.scrollbarImageCmd(),
	)
}

func (m Model) hasImageSurface() bool {
	cx := m.context
	if cx.images.graphics && cx.Editor.Options().Scrollbar {
		return true
	}
	if p, ok := m.compositor.activePreviewImager(); ok {
		if cx.images.graphics || p.hasPreviewImage(cx, m.compositor.size) {
			return true
		}
	}
	found := false
	cx.Editor.Tree().Range(func(p view.Pane) bool {
		_, found = p.(*ImagePane)
		return !found
	})
	return found
}

func (m Model) markImageDirty() {
	cx := m.context
	bars := cx.images.graphics && cx.Editor.Options().Scrollbar
	cx.Editor.Tree().Range(func(p view.Pane) bool {
		if _, ok := p.(*ImagePane); ok || bars {
			p.MarkDirty()
		}
		return true
	})
	if p, ok := m.compositor.activePreviewImager(); ok {
		p.markDirty()
	}
}

func workspaceTitle(dir string) string {
	root, _ := loader.FindWorkspace(dir)
	return underHome(root) + " - toe"
}

func underHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(home, path)
	if err != nil || !filepath.IsLocal(rel) {
		return path
	}
	return filepath.Join("~", rel)
}

// a font size change arrives as a resize, so the answer is re-asked for
func cellSizeQueryCmd(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tea.WindowSizeMsg); !ok {
		return nil
	}
	return tea.Raw(ansi.WindowOp(requestCellSizeOp))
}
