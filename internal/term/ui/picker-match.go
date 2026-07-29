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
	narrowed := canNarrowQuery(p.list.query, q)
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
	p.list.scroll = 0
	p.preview.scroll = 0
	p.clampScroll()
}

func (p *Picker) rebuildMatches() {
	src, _ := p.source.(StaticPickerSource)
	out := p.list.matched[:0]
	for i := range p.list.items {
		item := &p.list.items[i]
		if src == nil {
			out = append(out, pickerMatch{item: item, itemIndex: i})
			continue
		}
		if m, ok := p.scoreItem(src, item, i); ok {
			out = append(out, m)
		}
	}
	p.list.matched = out
	p.sortMatches()
}

func (p *Picker) narrowMatches() {
	// a longer query matches a subset of what its prefix matched, so the
	// rows already filtered out cannot come back
	src, ok := p.source.(StaticPickerSource)
	if !ok {
		return
	}
	out := p.list.matched[:0]
	for _, prev := range p.list.matched {
		if m, ok := p.scoreItem(src, prev.item, prev.itemIndex); ok {
			out = append(out, m)
		}
	}
	p.list.matched = out
	p.sortMatches()
}

func (p *Picker) scoreItem(
	src StaticPickerSource, item *PickerItem, index int,
) (pickerMatch, bool) {
	key := pickerScoreKey{
		query: p.list.query,
		text:  item.columnText(p.source.MatchColumn()),
	}
	cached, ok := p.list.scores[key]
	if !ok {
		if res, matched := src.Match(key.query, item); matched {
			cached = &res
		}
		if canCacheQuery(key.query) {
			p.list.scores[key] = cached
		}
	}
	if cached == nil {
		return pickerMatch{}, false
	}
	return pickerMatch{
		item:      item,
		itemIndex: index,
		result:    *cached,
	}, true
}

func (p *Picker) sortMatches() {
	slices.SortFunc(p.list.matched, func(a, b pickerMatch) int {
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

func (p *Picker) selection() *PickerItem {
	if p.list.cursor >= 0 && p.list.cursor < len(p.list.matched) {
		return p.list.matched[p.list.cursor].item
	}
	return nil
}

func (p *Picker) moveBy(n int) {
	if len(p.list.matched) == 0 {
		return
	}
	p.list.cursor = min(max(p.list.cursor+n, 0), len(p.list.matched)-1)
}

func (p *Picker) pageDown() {
	p.moveBy(max(p.list.height, 1))
}

func (p *Picker) pageUp() {
	p.moveBy(-max(p.list.height, 1))
}

func (p *Picker) clampScroll() {
	p.list.scroll = listClampScroll(
		p.list.scroll, len(p.list.matched), p.list.height,
	)
}

func (p *Picker) scrollBy(delta int) {
	p.list.scroll = listScrollBy(
		p.list.scroll, len(p.list.matched), p.list.height, delta,
	)
}

// ensureCursorVisible scrolls the list the minimum amount needed to bring the
// selected row into view, used after keyboard navigation
func (p *Picker) ensureCursorVisible() {
	p.list.scroll = listEnsureCursorVisible(
		p.list.scroll, p.list.cursor, len(p.list.matched), p.list.height,
	)
}

func canCacheQuery(query string) bool {
	// `%` and `\` route parts of the query to other columns, so the match
	// column's text no longer decides the score
	return !strings.ContainsAny(query, `%\`)
}

func canNarrowQuery(prev, next string) bool {
	return prev != "" && strings.HasPrefix(next, prev) && canCacheQuery(next)
}
