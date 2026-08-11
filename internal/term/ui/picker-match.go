package ui

import (
	"cmp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (p *Picker) setQuery(q string) tea.Cmd {
	if q == p.list.query {
		return nil
	}
	narrowed := canNarrowQuery(canNarrowQueryArgs{prev: p.list.query, next: q})
	p.list.query = q
	p.clearPreviewCache()
	if _, ok := p.source.(DynamicPickerSource); ok {
		return p.dynamicTriggerCmd()
	}
	if narrowed {
		p.narrowMatches()
	} else {
		p.rebuildMatches()
	}
	p.resetCursor()
	return nil
}

func (p *Picker) refilter() {
	p.rebuildMatches()
	p.resetCursor()
}

func (p *Picker) resetCursor() {
	if p.list.query != "" {
		p.list.cursor = 0
	}
	if p.list.cursor >= len(p.list.matched) {
		p.list.cursor = max(0, len(p.list.matched)-1)
	}
	p.ensureSelectable()
	p.list.scroll = 0
	p.preview.vScroll = 0
	p.preview.hScroll = 0
	p.clampScroll()
}

func (p *Picker) rebuildMatches() {
	src, _ := p.source.(StaticPickerSource)
	out := p.list.matched[:0]
	if src == nil {
		p.list.matched = unscoredItems(p.list.items, 0, out)
	} else {
		match := p.prepareMatcher(src)
		p.list.matched = p.scoreItems(match, p.list.items, 0, out)
	}
	p.sortMatches()
	p.insertSections()
}

func (p *Picker) scoreItems(
	match PickerMatcher, items []*PickerItem, startIndex int, out []pickerMatch,
) []pickerMatch {
	for i, item := range items {
		if m, ok := p.scoreItem(match, item, startIndex+i); ok {
			out = append(out, m)
		}
	}
	return out
}

func (p *Picker) narrowMatches() {
	// a longer query matches a subset of what its prefix matched, so the
	// rows already filtered out cannot come back
	src, ok := p.source.(StaticPickerSource)
	if !ok {
		return
	}
	match := p.prepareMatcher(src)
	out := p.list.matched[:0]
	for _, prev := range p.list.matched {
		if m, ok := p.scoreItem(match, prev.item, prev.itemIndex); ok {
			out = append(out, m)
		}
	}
	p.list.matched = out
	p.sortMatches()
	p.insertSections()
}

func (p *Picker) insertSections() {
	p.list.matchedSections = 0
	if len(p.list.sections) == 0 {
		return
	}
	out := make([]pickerMatch, 0, len(p.list.matched)+len(p.list.sections))
	group := 0
	for i, m := range p.list.matched {
		if i == 0 || m.item.Group != group {
			if label := p.sectionFor(m.item.Group); label != nil {
				out = append(out, pickerMatch{item: label})
				p.list.matchedSections++
			}
		}
		group = m.item.Group
		out = append(out, m)
	}
	p.list.matched = out
}

func (p *Picker) sectionFor(group int) *PickerItem {
	for _, section := range p.list.sections {
		if section.Group == group {
			return section
		}
	}
	return nil
}

func (p *Picker) scoreItem(
	match PickerMatcher, item *PickerItem, index int,
) (pickerMatch, bool) {
	key := pickerScoreKey{
		query: p.list.query,
		text:  item.columnText(p.source.MatchColumn()),
	}
	cached, ok := p.list.scores[key]
	if !ok {
		if res, matched := match(item); matched {
			cached = &res
		}
		if canCacheQuery(key.query) {
			p.list.scores[key] = cached
		}
	}
	if cached == nil {
		return pickerMatch{}, false
	}
	return pickerMatch{item: item, itemIndex: index, result: *cached}, true
}

func (p *Picker) prepareMatcher(src StaticPickerSource) PickerMatcher {
	if src == nil {
		return nil
	}
	return src.PrepareMatcher(p.list.query)
}

func (p *Picker) sortMatches() {
	slices.SortFunc(p.list.matched, func(a, b pickerMatch) int {
		if c := cmp.Compare(a.item.Group, b.item.Group); c != 0 {
			return c
		}
		if c := cmp.Compare(b.result.Score, a.result.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.itemIndex, b.itemIndex)
	})
}

func (p *PickerItem) columnText(col int) string {
	if col >= 0 && col < len(p.Columns) {
		return p.Columns[col]
	}
	if col == 0 {
		key := p.SortKey
		if key != "" {
			return key
		}
	}
	return p.Display
}

func (p *Picker) matchedCount() int {
	return len(p.list.matched) - p.list.matchedSections
}

func (p *Picker) selection() *PickerItem {
	if p.list.cursor >= 0 && p.list.cursor < len(p.list.matched) {
		if item := p.list.matched[p.list.cursor].item; !item.Section {
			return item
		}
	}
	return nil
}

func (p *Picker) moveBy(n int) {
	if len(p.list.matched) == 0 {
		return
	}
	step := 1
	if n < 0 {
		step, n = -1, -n
	}
	cur := p.list.cursor
	for ; n > 0; n-- {
		next := p.nextSelectable(nextSelectableArgs{from: cur, step: step})
		if next == cur {
			break
		}
		cur = next
	}
	p.list.cursor = min(max(cur, 0), len(p.list.matched)-1)
	// an explicit move supersedes a restore still waiting on a refill
	p.load.wantSet = false
}

func (p *Picker) ensureSelectable() {
	if p.selection() != nil || len(p.list.matched) == 0 {
		return
	}
	next := p.nextSelectable(nextSelectableArgs{from: p.list.cursor, step: 1})
	if next != p.list.cursor {
		p.list.cursor = next
		return
	}
	p.list.cursor = p.nextSelectable(nextSelectableArgs{
		from: p.list.cursor,
		step: -1,
	})
}

type nextSelectableArgs struct {
	from int
	step int
}

func (p *Picker) nextSelectable(args nextSelectableArgs) int {
	step := args.step
	for i := args.from + step; i >= 0 && i < len(p.list.matched); i += step {
		if !p.list.matched[i].item.Section {
			return i
		}
	}
	return args.from
}

func (p *Picker) pageDown() {
	p.moveBy(max(p.list.height, 1))
}

func (p *Picker) pageUp() {
	p.moveBy(-max(p.list.height, 1))
}

func (p *Picker) clampScroll() {
	p.list.scroll = listScroll{
		scroll: p.list.scroll,
		count:  len(p.list.matched),
		rows:   p.list.height,
	}.clamped()
}

func (p *Picker) scrollBy(delta int) {
	p.list.scroll = listScroll{
		scroll: p.list.scroll,
		count:  len(p.list.matched),
		rows:   p.list.height,
	}.scrollBy(delta)
}

// ensureCursorVisible scrolls the list the minimum amount needed to bring the
// selected row into view, used after keyboard navigation
func (p *Picker) ensureCursorVisible() {
	p.list.scroll = listScroll{
		scroll: p.list.scroll,
		cursor: p.list.cursor,
		count:  len(p.list.matched),
		rows:   p.list.height,
	}.ensureCursorVisible()
}

func canCacheQuery(query string) bool {
	// `%` and `\` route parts of the query to other columns, so the match
	// column's text no longer decides the score
	return !strings.ContainsAny(query, `%\`)
}

type canNarrowQueryArgs struct {
	prev string
	next string
}

func canNarrowQuery(args canNarrowQueryArgs) bool {
	prev := args.prev
	next := args.next
	return prev != "" && strings.HasPrefix(next, prev) && canCacheQuery(next)
}

func unscoredItems(
	items []*PickerItem, startIndex int, out []pickerMatch,
) []pickerMatch {
	for i, item := range items {
		out = append(out, pickerMatch{item: item, itemIndex: startIndex + i})
	}
	return out
}
