package ui

import (
	"cmp"
	"slices"

	tea "charm.land/bubbletea/v2"
)

func (p *Picker) setQuery(q string) tea.Cmd {
	if q == p.query {
		return nil
	}
	p.query = q
	p.clearPreviewCache()
	if _, ok := p.source.(DynamicPickerSource); ok {
		return p.dynamicTriggerCmd()
	}
	p.refilter()
	return nil
}

func (p *Picker) refilter() {
	p.rebuildMatches()
	if p.query != "" {
		p.cursor = 0
	}
	if p.cursor >= len(p.matched) {
		p.cursor = max(0, len(p.matched)-1)
	}
	p.listScroll = 0
	p.previewScroll = 0
	p.clampScroll()
}

func (p *Picker) rebuildMatches() {
	src, _ := p.source.(StaticPickerSource)
	p.matched = p.matched[:0]
	for i := range p.items {
		item := &p.items[i]
		if src == nil {
			p.matched = append(p.matched, pickerMatch{item: item})
			continue
		}
		score, indices, ok := src.Match(p.query, *item)
		if !ok {
			continue
		}
		p.matched = append(p.matched, pickerMatch{item, score, indices})
	}
	slices.SortStableFunc(p.matched, func(a, b pickerMatch) int {
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
	if p.cursor >= 0 && p.cursor < len(p.matched) {
		return p.matched[p.cursor].item
	}
	return nil
}

func (p *Picker) moveBy(n int) {
	if len(p.matched) == 0 {
		return
	}
	p.cursor = min(max(p.cursor+n, 0), len(p.matched)-1)
}

func (p *Picker) pageDown() {
	p.moveBy(max(p.listHeight, 1))
}

func (p *Picker) pageUp() {
	p.moveBy(-max(p.listHeight, 1))
}

func (p *Picker) clampScroll() {
	p.listScroll = listClampScroll(
		p.listScroll, len(p.matched), p.listHeight,
	)
}

func (p *Picker) scrollBy(delta int) {
	p.listScroll = listScrollBy(
		p.listScroll, len(p.matched), p.listHeight, delta,
	)
}

// ensureCursorVisible scrolls the list the minimum amount needed to bring the
// selected row into view, used after keyboard navigation
func (p *Picker) ensureCursorVisible() {
	p.listScroll = listEnsureCursorVisible(
		p.listScroll, p.cursor, len(p.matched), p.listHeight,
	)
}
