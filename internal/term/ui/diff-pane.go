package ui

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
)

type (
	// OpenDiffMsg asks the UI to show Claude's proposed change as a read-only
	// diff pane for review. toe does not apply or gate it; Claude's own prompt
	// does that in the terminal
	OpenDiffMsg struct {
		TabName     string
		Path        string
		NewContents string
		Shown       chan<- bool
	}

	// CloseDiffMsg asks the UI to close a diff opened by OpenDiffMsg; an empty
	// TabName closes all of them
	CloseDiffMsg struct {
		TabName string
	}

	// diffEntry is one file's proposed change within the review pane
	diffEntry struct {
		tab   string
		path  string
		lang  string
		base  *previewDocEntry
		work  *previewDocEntry
		lines []diffPreviewLine
	}

	// diffRow is one visible row of the appended review: a file header, or one
	// diff line belonging to an entry
	diffRow struct {
		entry  *diffEntry
		dl     diffPreviewLine
		header bool
	}

	// DiffPane shows read-only diffs of the changes Claude proposed, every file
	// appended into one continuous scroll the git-diff-preview way: each file
	// gets a header, highlighted in its own language. It never takes focus
	DiffPane struct {
		id      view.Id
		area    geom.Area
		dirty   bool
		entries []*diffEntry
		rows    []diffRow
		vScroll int
	}
)

var (
	_ view.Pane  = (*DiffPane)(nil)
	_ PaneInput  = (*DiffPane)(nil)
	_ PaneCursor = (*DiffPane)(nil)
)

// HandleEvent pages between files and scrolls the diff when focused manually
func (p *DiffPane) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, bool) {
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case m.Code == tea.KeyTab && m.Mod&tea.ModShift != 0:
			p.jumpFile(-1)
		case m.Code == tea.KeyTab:
			p.jumpFile(1)
		case m.Code == tea.KeyDown || m.Text == "j":
			p.scroll(1)
		case m.Code == tea.KeyUp || m.Text == "k":
			p.scroll(-1)
		case m.Code == tea.KeyPgDown || m.Code == tea.KeyKpPgDown:
			p.scroll(max(p.area.Height, 1))
		case m.Code == tea.KeyPgUp || m.Code == tea.KeyKpPgUp:
			p.scroll(-max(p.area.Height, 1))
		default:
			return ignored(), false
		}
	case tea.MouseWheelMsg:
		n := max(cx.Editor.Options().ScrollLines, 1)
		switch m.Button {
		case tea.MouseWheelUp:
			p.scroll(-n)
		case tea.MouseWheelDown:
			p.scroll(n)
		default:
			return ignored(), false
		}
	default:
		return ignored(), false
	}
	return consumed(), true
}

// Cursor reports that a diff pane has no text cursor
func (p *DiffPane) Cursor(*Context) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

// ID returns the pane identifier
func (p *DiffPane) ID() view.Id { return p.id }

// SetID sets the pane identifier
func (p *DiffPane) SetID(id view.Id) { p.id = id }

// Area returns the screen rectangle assigned by the layout tree
func (p *DiffPane) Area() geom.Area { return p.area }

// SetArea sets the screen rectangle assigned by the layout tree
func (p *DiffPane) SetArea(a geom.Area) {
	if a == p.area {
		return
	}
	p.area = a
	p.dirty = true
}

// MarkDirty flags the pane as needing a repaint
func (p *DiffPane) MarkDirty() { p.dirty = true }

// ConsumeDirty reports and clears whether the pane changed
func (p *DiffPane) ConsumeDirty() bool {
	dirty := p.dirty
	p.dirty = false
	return dirty
}

// Mode reports the diff review mode, which has no keymap bindings so paging and
// scroll keys reach this pane when it is focused manually
func (p *DiffPane) Mode() view.Mode { return view.ModeDiff }

// Path returns the file of the topmost visible diff
func (p *DiffPane) Path() string {
	if p.vScroll < len(p.rows) {
		return p.rows[p.vScroll].entry.path
	}
	if len(p.entries) > 0 {
		return p.entries[0].path
	}
	return ""
}

// SaveSession does not persist transient diff panes across restarts
func (p *DiffPane) SaveSession(*view.SessionWriter) {}

// Split refuses to split a diff pane
func (p *DiffPane) Split() (view.Pane, error) { return nil, view.ErrNoView }

// Discard releases this diff pane
func (p *DiffPane) Discard() {}

// Shutdown releases external resources owned by this pane
func (p *DiffPane) Shutdown() {}

func (p *DiffPane) add(entry *diffEntry) {
	for i, e := range p.entries {
		if e.tab == entry.tab {
			p.entries[i] = entry
			p.rebuild()
			return
		}
	}
	p.entries = append(p.entries, entry)
	p.rebuild()
}

func (p *DiffPane) remove(tab string) {
	for i, e := range p.entries {
		if e.tab == tab {
			p.entries = append(p.entries[:i], p.entries[i+1:]...)
			break
		}
	}
	p.rebuild()
}

// rebuild flattens the entries into the appended row list
func (p *DiffPane) rebuild() {
	rows := make([]diffRow, 0)
	for _, e := range p.entries {
		rows = append(rows, diffRow{entry: e, header: true})
		for _, dl := range e.lines {
			rows = append(rows, diffRow{entry: e, dl: dl})
		}
	}
	p.rows = rows
	p.vScroll = min(p.vScroll, p.maxScroll())
	p.dirty = true
}

func (p *DiffPane) maxScroll() int {
	return max(len(p.rows)-max(p.area.Height, 1), 0)
}

func (p *DiffPane) scroll(rows int) {
	v := min(max(p.vScroll+rows, 0), p.maxScroll())
	if v != p.vScroll {
		p.vScroll = v
		p.dirty = true
	}
}

// jumpFile moves the viewport to the next or previous file header
func (p *DiffPane) jumpFile(delta int) {
	headers := make([]int, 0, len(p.entries))
	for i, r := range p.rows {
		if r.header {
			headers = append(headers, i)
		}
	}
	if len(headers) == 0 {
		return
	}
	target := 0
	if delta > 0 {
		for _, h := range headers {
			if h > p.vScroll {
				target = h
				break
			}
		}
	} else {
		for _, h := range headers {
			if h >= p.vScroll {
				break
			}
			target = h
		}
	}
	p.vScroll = min(target, p.maxScroll())
	p.dirty = true
}

func (ec *EditorComponent) handleOpenDiff(
	msg OpenDiffMsg,
) (EventResult, tea.Cmd) {
	e := ec.context.Editor
	if e.Tree().Maximized() &&
		(ec.diff == nil || e.Tree().Focus() != ec.diff.ID()) {
		msg.respond(false)
		return consumed(), nil
	}
	fc := currentContent(e, msg.Path)
	entry := newDiffEntry(newDiffEntryArgs{
		syntax:   ec.context.Syntax,
		tab:      msg.TabName,
		path:     msg.Path,
		lang:     fc.lang,
		old:      fc.text,
		proposed: msg.NewContents,
	})
	if ec.diff == nil {
		pane := &DiffPane{dirty: true}
		pane.add(entry)
		if !placeDiffPane(e, pane) {
			msg.respond(false)
			return consumed(), nil
		}
		ec.diff = pane
	} else {
		ec.diff.add(entry)
	}
	ec.requestRedraw()
	msg.respond(true)
	return consumed(), nil
}

func (ec *EditorComponent) handleCloseDiff(
	tabName string,
) (EventResult, tea.Cmd) {
	if ec.diff == nil {
		return consumed(), nil
	}
	if tabName == "" {
		ec.diff.entries = nil
	} else {
		ec.diff.remove(tabName)
	}
	if len(ec.diff.entries) == 0 {
		e := ec.context.Editor
		e.ClosePane(ec.diff.ID())
		ec.diff = nil
	} else {
		ec.diff.rebuild()
	}
	ec.requestRedraw()
	return consumed(), nil
}

func placeDiffPane(e *view.Editor, pane *DiffPane) bool {
	focus := e.Tree().Focus()
	if !e.SplitPane(pane, view.LayoutVertical) {
		return false
	}
	e.FocusPane(focus)
	return true
}

func (m OpenDiffMsg) respond(shown bool) {
	if m.Shown != nil {
		m.Shown <- shown
	}
}

type currentContentRes struct {
	text string
	lang string
}

// currentContent returns the file's current text and language, from the open
// buffer when it exists, else from disk with no detected language
func currentContent(e *view.Editor, path string) currentContentRes {
	if abs, err := filepath.Abs(path); err == nil {
		for _, d := range e.AllDocuments() {
			if d.Path() == abs {
				return currentContentRes{
					text: d.Text().String(),
					lang: d.Lang(),
				}
			}
		}
	}
	data, _ := os.ReadFile(path)
	return currentContentRes{text: string(data)}
}

type newDiffEntryArgs struct {
	syntax   *syntax.Cache
	tab      string
	path     string
	lang     string
	old      string
	proposed string
}

// newDiffEntry builds one file's diff entry, highlighted for its language
func newDiffEntry(args newDiffEntryArgs) *diffEntry {
	oldText := highlight.NormalizeNewlines(args.old)
	newText := highlight.NormalizeNewlines(args.proposed)
	oldRope := core.NewRope(oldText)
	newRope := core.NewRope(newText)
	entry := func(text string, rope core.Rope) *previewDocEntry {
		return &previewDocEntry{
			lang: args.lang,
			rope: rope,
			spans: previewSpans(previewSpansArgs{
				cache: args.syntax, text: text, lang: args.lang,
			}),
		}
	}
	lines := buildDiffPreviewLines(buildDiffPreviewLinesArgs{
		kind:    view.FileChangeModified,
		working: newRope,
		base:    oldRope,
		hunks:   view.Diff(view.DiffSides{Base: oldRope, Doc: newRope}),
	})
	return &diffEntry{
		tab:   args.tab,
		path:  args.path,
		lang:  args.lang,
		base:  entry(oldText, oldRope),
		work:  entry(newText, newRope),
		lines: lines,
	}
}
