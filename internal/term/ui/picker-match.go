package ui

import (
	"cmp"
	"slices"

	tea "charm.land/bubbletea/v2"
)

func (p *Picker) setQuery(q string) tea.Cmd {
	if q == p.list.query {
		return nil
	}
	p.list.query = q
	p.clearPreviewCache()
	if _, ok := p.source.(DynamicPickerSource); ok {
		return p.dynamicTriggerCmd()
	}
	p.refilter()
	return nil
}

func (p *Picker) refilter() {
	p.rebuildMatches()
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
	p.list.matched = p.list.matched[:0]
	for i := range p.list.items {
		item := &p.list.items[i]
		if src == nil {
			p.list.matched = append(p.list.matched, pickerMatch{item: item})
			continue
		}
		score, indices, ok := src.Match(p.list.query, *item)
		if !ok {
			continue
		}
		p.list.matched = append(
			p.list.matched, pickerMatch{
				item:    item,
				score:   score,
				indices: indices,
			},
		)
	}
	slices.SortStableFunc(p.list.matched, func(a, b pickerMatch) int {
		return cmp.Compare(b.score, a.score)
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
