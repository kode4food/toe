package ui

import "github.com/mattn/go-runewidth"

func pickerHasHeader(cols []string) bool {
	if len(cols) <= 1 {
		return false
	}
	for _, col := range cols {
		if col != "" {
			return true
		}
	}
	return false
}

func pickerColumnWidths(p *Picker, w int) []int {
	cols := p.source.Columns()
	n := len(cols)
	widths := make([]int, n)
	if n == 0 {
		return widths
	}
	spacing := max(n-1, 0)
	available := max(w-spacing, 0)
	proportions := p.source.ColumnProportions()
	if len(proportions) != n {
		proportions = defaultColumnProportions(n)
	}
	for i, col := range cols {
		widths[i] = runewidth.StringWidth(col)
	}
	for _, m := range p.matched {
		for i := range min(len(m.item.Columns), n) {
			widths[i] = max(widths[i], runewidth.StringWidth(m.item.Columns[i]))
		}
	}
	return pickerProportionalColumnWidths(widths, proportions, available)
}

func pickerProportionalColumnWidths(
	measured []int, proportions []int, available int,
) []int {
	widths := make([]int, len(measured))
	weight := 0
	pinned := 0
	for i, proportion := range proportions {
		if proportion <= 0 {
			widths[i] = measured[i]
			pinned += widths[i]
			continue
		}
		weight += proportion
	}
	remaining := max(available-pinned, 0)
	used := 0
	if weight > 0 {
		for i, proportion := range proportions {
			if proportion <= 0 {
				continue
			}
			widths[i] = remaining * proportion / weight
			used += widths[i]
		}
		for i, proportion := range proportions {
			if used >= remaining {
				break
			}
			if proportion <= 0 {
				continue
			}
			widths[i]++
			used++
		}
	}
	total := 0
	for _, width := range widths {
		total += width
	}
	for i := len(widths) - 1; total > available && i >= 0; i-- {
		take := min(widths[i], total-available)
		widths[i] -= take
		total -= take
	}
	return widths
}
