package action

import (
	"regexp"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// SelectWithinRegex keeps only the parts of each selection that match regex
func SelectWithinRegex(e *view.Editor, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	var out []core.Range
	primary := 0
	for i, r := range sel.Ranges() {
		from, to := r.From(), r.To()
		slice, err := text.Slice(from, to)
		if err != nil {
			continue
		}
		s := slice.String()
		for _, loc := range re.FindAllStringIndex(s, -1) {
			start := from + byteOffsetToRuneOffset(s, loc[0])
			end := from + byteOffsetToRuneOffset(s, loc[1])
			out = append(out, core.NewRange(start, end))
		}
		if i == sel.PrimaryIndex() && len(out) > 0 {
			primary = len(out) - 1
		}
	}
	if len(out) == 0 {
		return nil
	}
	newSel, err := core.NewSelection(out, primary)
	if err != nil {
		return err
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return nil
}

// SplitSelectionByRegex splits each selection range at every regex match
func SplitSelectionByRegex(e *view.Editor, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	var out []core.Range
	primary := 0
	for i, r := range sel.Ranges() {
		from, to := r.From(), r.To()
		slice, err := text.Slice(from, to)
		if err != nil {
			continue
		}
		s := slice.String()
		indices := re.FindAllStringIndex(s, -1)
		prev := 0
		for _, loc := range indices {
			if loc[0] > prev {
				out = append(out, core.NewRange(
					from+byteOffsetToRuneOffset(s, prev),
					from+byteOffsetToRuneOffset(s, loc[0]),
				))
			}
			prev = loc[1]
		}
		if prev < len(s) {
			out = append(out, core.NewRange(
				from+byteOffsetToRuneOffset(s, prev), to,
			))
		}
		if i == sel.PrimaryIndex() && len(out) > 0 {
			primary = len(out) - 1
		}
	}
	if len(out) == 0 {
		return nil
	}
	newSel, err := core.NewSelection(out, primary)
	if err != nil {
		return err
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return nil
}

// KeepSelectionsMatching keeps only selection ranges whose text matches regex
func KeepSelectionsMatching(e *view.Editor, pattern string) error {
	return filterSelectionsImpl(e, pattern, false)
}

// RemoveSelectionsMatching removes selection ranges whose text matches regex
func RemoveSelectionsMatching(e *view.Editor, pattern string) error {
	return filterSelectionsImpl(e, pattern, true)
}

func filterSelectionsImpl(e *view.Editor, pattern string, remove bool) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	var out []core.Range
	primary := 0
	for i, r := range sel.Ranges() {
		slice, err := text.Slice(r.From(), r.To())
		if err != nil {
			continue
		}
		if re.MatchString(slice.String()) != remove {
			out = append(out, r)
			if i == sel.PrimaryIndex() {
				primary = len(out) - 1
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	newSel, err := core.NewSelection(out, primary)
	if err != nil {
		return err
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return nil
}
