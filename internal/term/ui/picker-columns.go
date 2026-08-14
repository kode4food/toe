package ui

import (
	"slices"

	"github.com/mattn/go-runewidth"
)

type pickerColumnSizing struct {
	widths      []int
	minimum     []int
	preferred   []int
	maximum     []int
	proportions []int
	spare       int
}

const (
	pickerColumnMinWidth   = 3
	pickerColumnPercentile = 90
)

func newPickerColumnSizing(p *Picker, w int) *pickerColumnSizing {
	cols := p.source.Columns()
	n := len(cols)
	spacing := max(n-1, 0)
	proportions := p.source.ColumnProportions()
	if len(proportions) != n {
		proportions = defaultColumnProportions(n)
	}
	s := &pickerColumnSizing{
		widths:      make([]int, n),
		minimum:     make([]int, n),
		preferred:   make([]int, n),
		maximum:     make([]int, n),
		proportions: proportions,
		spare:       max(w-spacing, 0),
	}
	s.measure(p, cols)
	return s
}

func (s *pickerColumnSizing) measure(p *Picker, cols []string) {
	for i, col := range cols {
		header := runewidth.StringWidth(col)
		s.maximum[i] = header
		measured := make([]int, 0, len(p.list.matched))
		for _, m := range p.list.matched {
			if i >= len(m.item.Columns) {
				continue
			}
			width := runewidth.StringWidth(m.item.Columns[i])
			if i == pickerFileIconColumn(p, m.item) {
				width = max(
					width, runewidth.StringWidth(pickerDefaultFileIcon.glyph),
				)
			}
			s.maximum[i] = max(s.maximum[i], width)
			measured = append(measured, width)
		}
		s.minimum[i] = header
		if col == "" {
			s.minimum[i] = min(s.maximum[i], pickerColumnMinWidth)
		}
		s.preferred[i] = s.minimum[i]
		if len(measured) > 0 {
			slices.Sort(measured)
			idx := (len(measured)*pickerColumnPercentile - 1) / 100
			s.preferred[i] = max(s.preferred[i], measured[idx])
		}
	}
}

func (s *pickerColumnSizing) allocate() {
	s.grow(s.minimum, true)
	s.grow(s.minimum, false)
	s.grow(s.preferred, true)
	s.grow(s.preferred, false)
	s.grow(s.maximum, true)
	s.grow(s.maximum, false)
}

func (s *pickerColumnSizing) grow(target []int, pinned bool) {
	for s.spare > 0 {
		best := -1
		bestWeight := 1
		for i, proportion := range s.proportions {
			if (proportion <= 0) != pinned || s.widths[i] >= target[i] {
				continue
			}
			weight := max(proportion, 1)
			if best < 0 ||
				s.widths[i]*bestWeight < s.widths[best]*weight {
				best = i
				bestWeight = weight
			}
		}
		if best < 0 {
			break
		}
		s.widths[best]++
		s.spare--
	}
}

func pickerColumnWidths(p *Picker, w int) []int {
	s := newPickerColumnSizing(p, w)
	s.allocate()
	return s.widths
}

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
