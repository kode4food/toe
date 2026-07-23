package ui

import (
	"cmp"
	"errors"
	"slices"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
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

		previewCache  previewCache
		diffBaseCache map[string]diffBaseResult

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
		ID() string
		Columns() []string
		MatchColumn() int
		ColumnProportions() []int
		Load(*view.Editor) ([]PickerItem, <-chan PickerItem, StopFunc)
		Accept(*view.Editor, PickerItem, PickerAcceptAction)
	}

	// PickerPreviewSkipper marks picker sources that never render previews
	PickerPreviewSkipper interface {
		SkipPreview()
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
		Columns     []string
		StyleScopes []string
		SortKey     string
		Preview     PreviewRenderer
		Location    PickerLocation
		Payload     any
		DiffHunks   []view.DiffHunk
		DiffPreview bool
		DiffKind    view.FileChangeKind
		BasePath    string
	}

	// PreviewRenderer renders a picker item's preview at the given size
	PreviewRenderer func(geom.Size) string

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

	// PickerBase is an optional starting point a source can embed for default
	// id, column, and fuzzy-match behavior; a source is free to implement
	// those methods itself instead
	PickerBase struct {
		id          string
		columns     []string
		matchColumn int
		proportions []int
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
		source:        source,
		previewCache:  previewCache{},
		diffBaseCache: map[string]diffBaseResult{},
		cancel:        func() {},
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

// NewPickerBase builds the fixed metadata a source embeds: kebab-case id,
// column headers, the column matched against, and each column's flex weight
func NewPickerBase(
	id string, columns []string, matchColumn int, proportions []int,
) PickerBase {
	return PickerBase{
		id:          id,
		columns:     columns,
		matchColumn: matchColumn,
		proportions: proportions,
	}
}

func (p PickerBase) ID() string {
	return p.id
}

func (p PickerBase) Columns() []string {
	return p.columns
}

func (p PickerBase) MatchColumn() int {
	return p.matchColumn
}

func (p PickerBase) ColumnProportions() []int {
	if len(p.proportions) == len(p.columns) {
		for _, proportion := range p.proportions {
			if proportion > 0 {
				return p.proportions
			}
		}
	}
	return defaultColumnProportions(len(p.columns))
}

func (p PickerBase) Match(query string, item PickerItem) (int, []int, bool) {
	return fuzzyMatchItem(query, item, p.columns, p.matchColumn)
}

// MatchCount reports how many items currently match the query
func (p *Picker) MatchCount() int {
	return len(p.matched)
}

// SelectIndex moves the cursor to i when it is a valid match index
func (p *Picker) SelectIndex(i int) {
	if i >= 0 && i < len(p.matched) {
		p.cursor = i
	}
}

func (p *Picker) addItems(items []PickerItem) {
	if len(items) == 0 {
		return
	}
	target, hadSelection := p.selectedTarget()
	p.items = append(p.items, items...)
	SortPickerItems(p.items)
	p.rebuildMatches()
	if hadSelection {
		p.restoreSelection(target)
	} else if p.cursor >= len(p.matched) {
		p.cursor = max(0, len(p.matched)-1)
	}
	p.clampScroll()
}

func (p *Picker) selectedTarget() (PickerTarget, bool) {
	if p.cursor < 0 || p.cursor >= len(p.matched) {
		return PickerTarget{}, false
	}
	target := p.matched[p.cursor].item.Location.Target
	return target, target.Valid()
}

func (p *Picker) restoreSelection(target PickerTarget) {
	for i, m := range p.matched {
		if m.item.Location.Target == target {
			p.cursor = i
			return
		}
	}
	if p.cursor >= len(p.matched) {
		p.cursor = max(0, len(p.matched)-1)
	}
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
	clear(p.diffBaseCache)
}

// AcceptDocumentID opens the document by id, splitting per action, and
// returns the view now showing it
func AcceptDocumentID(
	e *view.Editor, id view.DocumentId, action PickerAcceptAction,
) (*view.View, bool) {
	switch action {
	case PickerAcceptHorizontalSplit:
		return e.HSplit(id)
	case PickerAcceptVerticalSplit:
		return e.VSplit(id)
	default:
		return e.ShowDocument(id)
	}
}

// OpenPath opens a text document or image pane at path
func OpenPath(
	e *view.Editor, path string, action PickerAcceptAction,
) (*view.View, bool, error) {
	if path == "" {
		return nil, false, view.ErrDocumentNoPath
	}
	doc, err := e.SwitchOrOpenDoc(path)
	if err == nil {
		v, ok := AcceptDocumentID(e, doc.ID(), action)
		return v, ok, nil
	}
	if !errors.Is(err, core.ErrBinaryFile) {
		return nil, false, err
	}
	pane, err := NewImagePane(e, path)
	if err != nil {
		return nil, false, err
	}
	if !acceptImagePane(e, pane, action) {
		return nil, false, view.ErrNoView
	}
	return nil, true, nil
}

// AcceptPath opens the file at path (switching to it if already open),
// splitting per action, and returns the view now showing it
func AcceptPath(
	e *view.Editor, path string, action PickerAcceptAction,
) (*view.View, bool) {
	if path == "" {
		return nil, false
	}
	v, ok, err := OpenPath(e, path, action)
	if err != nil {
		e.SetStatusMsg(i18n.Text(i18n.ErrorMessage, i18n.Vars{
			"message": err,
		}))
		return nil, false
	}
	return v, ok
}

// AlignAcceptedView scrolls the view so the accepted document's cursor is
// visible after a picker jump
func AlignAcceptedView(e *view.Editor, v *view.View, doc *view.Document) {
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

func acceptImagePane(
	e *view.Editor, pane *ImagePane, action PickerAcceptAction,
) bool {
	switch action {
	case PickerAcceptHorizontalSplit:
		if !e.SplitPane(pane, view.LayoutHorizontal) {
			return false
		}
	case PickerAcceptVerticalSplit:
		if !e.SplitPane(pane, view.LayoutVertical) {
			return false
		}
	default:
		id := e.Tree().Focus()
		if e.Tree().Get(id) == nil {
			return false
		}
		old := e.ReplacePane(id, pane)
		e.DiscardPane(old)
	}
	return true
}

func defaultColumnProportions(n int) []int {
	proportions := make([]int, n)
	if n > 0 {
		proportions[0] = 1
	}
	return proportions
}

func previewEnabled(source PickerSource) bool {
	_, skip := source.(PickerPreviewSkipper)
	return !skip
}

// SortPickerItems sorts items by display text, the default ordering for
// static picker sources
func SortPickerItems(items []PickerItem) {
	slices.SortStableFunc(items, func(a, b PickerItem) int {
		return cmp.Compare(a.Display, b.Display)
	})
}
