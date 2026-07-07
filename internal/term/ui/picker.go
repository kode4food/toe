package ui

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/view"
)

type (
	// Picker holds the runtime state for an open picker overlay
	Picker struct {
		source PickerSource

		items   []PickerItem
		matched []pickerMatch
		query   string

		cursor     int
		listScroll int
		listHeight int

		previewScroll    int
		previewScrollFor int

		previewCache previewCache

		feedCmd tea.Cmd
		cancel  StopFunc

		dynamicGen     int
		dynamicStop    StopFunc
		dynamicPending bool
	}

	// PickerFunc constructs a Picker from the editor
	PickerFunc func(e *view.Editor) *Picker

	// StopFunc cancels an in-progress feed or search
	StopFunc func()

	// PickerSource is implemented by every picker data source
	PickerSource interface {
		Title() string
		Columns() []string
		Primary() int
		Load(*view.Editor) ([]PickerItem, <-chan PickerItem, StopFunc)
		Accept(*view.Editor, PickerItem, PickerAcceptAction)
	}

	// StaticPickerSource extends PickerSource with fuzzy-match filtering
	StaticPickerSource interface {
		PickerSource
		Match(query string, item PickerItem) (score int, indices []int, ok bool)
	}

	// DynamicPickerSource extends PickerSource with query-driven search
	DynamicPickerSource interface {
		PickerSource
		Search(query string)
	}

	// NavigablePickerSource extends PickerSource for pickers that can drill
	// into sub-pickers. Navigate returns a PickerFunc to replace the current
	// picker, or nil to fall through to Accept
	NavigablePickerSource interface {
		PickerSource
		Navigate(*view.Editor, PickerItem) PickerFunc
	}

	// PickerItem is a single row shown in the picker list
	PickerItem struct {
		Display     string
		Style       lipgloss.Style
		Columns     []string
		StyleScopes []string
		SortKey     string
		Preview     PreviewRenderer
		Location    PickerLocation
		Payload     any
		DiffHunks   []view.DiffHunk
	}

	// PreviewRenderer renders a picker item's preview at the given size
	PreviewRenderer func(w, h int) string

	// PickerLocation holds a target and an optional line range
	PickerLocation struct {
		Target PickerTarget
		Lines  *PickerLineRange
	}

	PickerLineRange struct {
		From int
		To   int
	}

	// PickerTarget identifies a document by path or in-memory ID
	PickerTarget struct {
		Path string
		ID   view.DocumentId
	}

	PickerAcceptAction int

	pickerMatch struct {
		item    *PickerItem
		score   int
		indices []int
	}

	pickerMeta struct {
		title   string
		columns []string
		primary int
	}

	pickerDynamicTriggerMsg struct {
		gen   int
		query string
	}
)

const pickerDynamicDelay = 275 * time.Millisecond

const (
	PickerAcceptReplace PickerAcceptAction = iota
	PickerAcceptHorizontalSplit
	PickerAcceptVerticalSplit
)

// NewPicker constructs a Picker for the given source, triggering Load
// immediately. The returned feedCmd (if any) must be dispatched by the
// caller after mounting the component
func NewPicker(e *view.Editor, source PickerSource) *Picker {
	p := &Picker{
		source:       source,
		previewCache: previewCache{},
		cancel:       func() {},
	}
	items, feed, stop := source.Load(e)
	p.cancel = stop
	_, isStatic := source.(StaticPickerSource)
	p.items = items
	if isStatic {
		p.refilter()
	} else {
		p.matched = make([]pickerMatch, len(items))
		for i := range items {
			p.matched[i] = pickerMatch{item: &items[i]}
		}
	}
	if feed != nil {
		done := make(chan struct{})
		closeDone := sync.OnceFunc(func() { close(done) })
		oldStop := p.cancel
		p.cancel = func() { oldStop(); closeDone() }
		p.feedCmd = drainPickerFeed(feed, done)
	}
	return p
}

// Valid reports whether the target refers to a real document or path
func (p PickerTarget) Valid() bool {
	return p.Path != "" || p.ID != view.InvalidDocumentId
}

func (p pickerMeta) Title() string {
	return p.title
}

func (p pickerMeta) Columns() []string {
	return p.columns
}

func (p pickerMeta) Primary() int {
	return p.primary
}

func (p pickerMeta) Match(query string, item PickerItem) (int, []int, bool) {
	return fuzzyMatchItem(query, item, p.columns, p.primary)
}

func (p *Picker) addItems(items []PickerItem) {
	if len(items) == 0 {
		return
	}
	p.items = append(p.items, items...)
	sortPickerItems(p.items)
	p.refilter()
}

func (p *Picker) addDynamicItems(items []PickerItem) {
	if len(items) == 0 {
		return
	}
	p.items = append(p.items, items...)
	p.matched = make([]pickerMatch, len(p.items))
	for i := range p.items {
		p.matched[i] = pickerMatch{item: &p.items[i]}
	}
	if p.cursor >= len(p.matched) {
		p.cursor = max(0, len(p.matched)-1)
	}
}

func (p *Picker) dynamicTriggerCmd() tea.Cmd {
	_, isDynamic := p.source.(DynamicPickerSource)
	if !isDynamic {
		return nil
	}
	if p.dynamicStop != nil {
		p.dynamicStop()
		p.dynamicStop = nil
	}
	p.items = nil
	p.matched = nil
	p.cursor = 0
	p.listScroll = 0
	p.previewScroll = 0
	p.dynamicGen++
	gen := p.dynamicGen
	q := p.query
	if q == "" {
		p.dynamicPending = false
		return nil
	}
	p.dynamicPending = true
	return func() tea.Msg {
		time.Sleep(pickerDynamicDelay)
		return pickerDynamicTriggerMsg{gen: gen, query: q}
	}
}

func (p *Picker) clearPreviewCache() {
	clear(p.previewCache)
}

func acceptDocumentID(
	e *view.Editor, id view.DocumentId, action PickerAcceptAction,
) (*view.View, bool) {
	switch action {
	case PickerAcceptHorizontalSplit:
		return e.HSplit(id)
	case PickerAcceptVerticalSplit:
		return e.VSplit(id)
	default:
		return replaceDocumentID(e, id)
	}
}

func replaceDocumentID(e *view.Editor, id view.DocumentId) (*view.View, bool) {
	for _, v := range e.AllViews() {
		if v.DocID() != id {
			continue
		}
		e.FocusView(v.ID())
		return v, true
	}
	if !e.SwitchBuffer(id) {
		return nil, false
	}
	return e.FocusedView()
}

func acceptPath(
	e *view.Editor, path string, action PickerAcceptAction,
) (*view.View, bool) {
	if path == "" {
		return nil, false
	}
	doc, err := e.SwitchOrOpenDoc(path)
	if err != nil {
		return nil, false
	}
	return acceptDocumentID(e, doc.ID(), action)
}

func alignAcceptedView(e *view.Editor, v *view.View, doc *view.Document) {
	sel := doc.SelectionFor(v.ID())
	v.EnsureCursorVisible(
		doc.Text(), sel, max(v.Area().Height, e.ViewHeight()),
		e.Options().ScrollOff, nil,
	)
	v.EnsureCursorVisibleHorizontal(
		doc.Text(), sel, e.ViewContentWidth(), doc.TabWidth(),
		e.Options().ScrollOff,
	)
}
