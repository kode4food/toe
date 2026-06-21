package ui

import (
	"cmp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (p *Picker) setQuery(q string) tea.Cmd {
	if q == p.query {
		return nil
	}
	p.query = q
	if _, ok := p.source.(DynamicPickerSource); ok {
		return p.dynamicTriggerCmd()
	}
	p.refilter()
	return nil
}

func (p *Picker) refilter() {
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
	if p.query != "" {
		p.cursor = 0
	}
	if p.cursor >= len(p.matched) {
		p.cursor = max(0, len(p.matched)-1)
	}
	// a fresh result set views from the top; clamp handles a shrunk list
	p.listScroll = 0
	p.previewScroll = 0
	p.clampScroll()
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
	n = ((n % len(p.matched)) + len(p.matched)) % len(p.matched)
	p.cursor = (p.cursor + n) % len(p.matched)
}

func (p *Picker) pageDown() {
	p.moveBy(max(p.listHeight, 1))
}

func (p *Picker) pageUp() {
	p.moveBy(-max(p.listHeight, 1))
}

// maxScroll is the largest valid first-visible row, leaving the last page of
// items filling the viewport
func (p *Picker) maxScroll() int {
	return max(0, len(p.matched)-max(p.listHeight, 1))
}

// clampScroll keeps the scroll offset within the valid range
func (p *Picker) clampScroll() {
	p.listScroll = max(0, min(p.listScroll, p.maxScroll()))
}

// scrollBy moves the list view by delta rows without changing the selection
func (p *Picker) scrollBy(delta int) {
	p.listScroll += delta
	p.clampScroll()
}

// ensureCursorVisible scrolls the list the minimum amount needed to bring the
// selected row into view, used after keyboard navigation
func (p *Picker) ensureCursorVisible() {
	h := max(p.listHeight, 1)
	if p.cursor < p.listScroll {
		p.listScroll = p.cursor
	} else if p.cursor >= p.listScroll+h {
		p.listScroll = p.cursor - h + 1
	}
	p.clampScroll()
}

func fuzzyMatchItem(
	query string, item PickerItem, columns []string, primary int,
) (int, []int, bool) {
	fields := parsePickerQuery(columns, primary, query)
	score := 0
	var indices []int
	for col, pat := range fields {
		key := item.columnText(col)
		s, idx := fuzzyMatch(strings.ToLower(pat), key)
		if s < 0 {
			return 0, nil, false
		}
		score += s
		if col == primary {
			indices = idx
		}
	}
	return score, indices, true
}

func parsePickerQuery(
	columns []string, primary int, input string,
) map[int]string {
	fields := map[int]string{}
	if input == "" {
		fields[primary] = ""
		return fields
	}
	field := primary
	var fieldText strings.Builder
	var text strings.Builder
	escaped := false
	inField := false
	finish := func() {
		pat := strings.TrimSuffix(text.String(), " ")
		if pat != "" {
			if prev := fields[field]; prev != "" {
				fields[field] = prev + " " + pat
			} else {
				fields[field] = pat
			}
		}
		text.Reset()
	}
	for _, ch := range input {
		switch {
		case escaped:
			if ch != '%' {
				text.WriteRune('\\')
			}
			text.WriteRune(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case ch == '%':
			if text.Len() > 0 {
				finish()
			}
			field = primary
			fieldText.Reset()
			inField = true
		case ch == ' ' && inField:
			text.Reset()
			inField = false
		case inField:
			fieldText.WriteRune(ch)
			if idx, ok := matchPickerColumn(columns, fieldText.String()); ok {
				field = idx
			}
		default:
			text.WriteRune(ch)
		}
	}
	if !inField && text.Len() > 0 {
		finish()
	}
	if len(fields) == 0 {
		fields[primary] = ""
	}
	return fields
}

func matchPickerColumn(columns []string, prefix string) (int, bool) {
	best := -1
	for i, col := range columns {
		if !strings.HasPrefix(col, prefix) {
			continue
		}
		if best < 0 || len(col) < len(columns[best]) {
			best = i
		}
	}
	return best, best >= 0
}

func fuzzyMatch(pat, text string) (int, []int) {
	if len(pat) == 0 {
		return 0, nil
	}
	pr := []rune(pat)
	tr := []rune(text)
	tl := []rune(strings.ToLower(text))
	if len(pr) > len(tr) {
		return -1, nil
	}

	indices := make([]int, 0, len(pr))
	j := 0
	for i, c := range tl {
		if j < len(pr) && c == pr[j] {
			indices = append(indices, i)
			j++
		}
	}
	if j < len(pr) {
		return -1, nil
	}

	score := 0
	prev := -2
	for _, idx := range indices {
		if prev >= 0 && idx == prev+1 {
			score += 5
		}
		switch idx {
		case 0:
			score += 10
		default:
			switch tr[idx-1] {
			case '/', '\\', '.', '-', '_', ' ':
				score += 8
			}
		}
		prev = idx
	}
	score -= len(tr) / 4
	return score, indices
}
