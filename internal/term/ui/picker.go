package ui

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Picker holds the runtime state for an open picker overlay
	Picker struct {
		source  PickerSource
		items   []PickerItem
		matched []pickerMatch
		query   string
		cursor  int
		// listScroll is the first visible row of the list, decoupled from the
		// cursor so the wheel can scroll without changing the selection
		listScroll int
		// previewScroll offsets the preview viewport by logical lines on top of
		// the match anchor; the wheel adjusts it and the renderer clamps it.
		// previewScrollFor is the cursor index it applies to, so it resets to 0
		// whenever the selection changes
		previewScroll    int
		previewScrollFor int
		// listHeight is the number of result rows visible in the list pane,
		// computed during render and used for paging and scroll clamping
		listHeight int
		preview    bool
		spanCache  map[view.DocumentId]previewSpanEntry
		// fileCache caches the rope and syntax spans for file-path previews,
		// avoiding re-tokenization across frames (geometry-independent, valid
		// until the picker is closed)
		fileCache   map[string]fileSpanEntry
		feedCmd     tea.Cmd
		cancel      StopFunc
		dynamicGen  int
		dynamicStop StopFunc
	}

	// previewSpanEntry caches the materialized rope and syntax spans for an
	// in-memory document preview, reused across frames until the document's
	// revision or language changes
	previewSpanEntry struct {
		rev   int
		lang  string
		rope  core.Rope
		spans []highlight.Span
	}

	// fileSpanEntry caches the rope and syntax spans for a file-path preview,
	// keyed by path. Unlike previewSpanEntry it has no revision since the file
	// is read once and treated as stable for the lifetime of the picker
	fileSpanEntry struct {
		rope  core.Rope
		spans []highlight.Span
		lang  string
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
		Accept(*view.Editor, PickerItem)
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
		Display  string
		Style    lipgloss.Style
		Columns  []string
		SortKey  string
		Preview  func(w, h int) string
		Location PickerLocation
		Payload  any
	}

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
)

const pickerDynamicDelay = 275 * time.Millisecond

// NewPicker constructs a Picker for the given source, triggering Load
// immediately. The returned feedCmd (if any) must be dispatched by the
// caller after mounting the component
func NewPicker(e *view.Editor, source PickerSource) *Picker {
	p := &Picker{
		source:    source,
		spanCache: map[view.DocumentId]previewSpanEntry{},
		fileCache: map[string]fileSpanEntry{},
		cancel:    func() {},
	}
	items, feed, stop := source.Load(e)
	p.cancel = stop
	_, isStatic := source.(StaticPickerSource)
	_, isDynamic := source.(DynamicPickerSource)
	p.items = items
	p.preview = isDynamic || hasPreview(items)
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

func (p *Picker) addItems(items []PickerItem) {
	if len(items) == 0 {
		return
	}
	p.items = append(p.items, items...)
	sortPickerItems(p.items)
	p.preview = hasPreview(p.items)
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
		return nil
	}
	return func() tea.Msg {
		time.Sleep(pickerDynamicDelay)
		return pickerDynamicTriggerMsg{gen: gen, query: q}
	}
}

func (p PickerLocation) lineRange() (int, int, bool) {
	if p.Lines == nil {
		return 0, 0, false
	}
	return p.Lines.From, p.Lines.To, true
}
