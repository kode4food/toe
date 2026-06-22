package action

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// SortSelections sorts the text of each selection range lexicographically,
// replacing each range with the sorted text. Mirrors :sort in the reference
func SortSelections(e *view.Editor, reverse, insensitive bool) error {
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
	ranges := sel.Ranges()
	if len(ranges) < 2 {
		return errors.New(
			"sorting requires multiple selections; hint: split selection first")
	}

	fragments := make([]string, len(ranges))
	for i, r := range ranges {
		start, end := r.From(), r.To()
		if start > end {
			start, end = end, start
		}
		sl, _ := text.Slice(start, end)
		fragments[i] = sl.String()
	}

	cmp := func(a, b string) int {
		ka, kb := a, b
		if insensitive {
			ka = strings.ToLower(a)
			kb = strings.ToLower(b)
		}
		if ka < kb {
			return -1
		}
		if ka > kb {
			return 1
		}
		return 0
	}
	slices.SortStableFunc(fragments, func(a, b string) int {
		c := cmp(a, b)
		if reverse {
			return -c
		}
		return c
	})

	changes := make([]core.Change, len(ranges))
	for i, r := range ranges {
		start, end := r.From(), r.To()
		if start > end {
			start, end = end, start
		}
		changes[i] = core.TextChange(start, end, fragments[i])
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return err
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return err
	}
	return e.Apply(
		core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// ReflowSelections reflows the text of each selection range to the given
// column width. Mirrors :reflow in the reference
func ReflowSelections(e *view.Editor, width int) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		start, end := r.From(), r.To()
		if start > end {
			start, end = end, start
		}
		fragRope, _ := text.Slice(start, end)
		reflowed := core.ReflowHardWrap(fragRope.String(), width)
		changes = append(changes, core.TextChange(start, end, reflowed))
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(
		core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// CharInfo returns a string describing the grapheme at the primary cursor.
// Format: "<printable>" (U+XXXX ...) [Dec N] Hex xx [+xx ...]
func CharInfo(e *view.Editor) string {
	v, ok := e.FocusedView()
	if !ok {
		return ""
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return ""
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	start := sel.Primary().Cursor(text)
	end := core.NextGraphemeBoundary(text, start)
	if start == end {
		return ""
	}
	gr, err := text.Slice(start, end)
	if err != nil {
		return ""
	}
	grapheme := gr.String()

	var printable strings.Builder
	for _, c := range grapheme {
		switch c {
		case '\000':
			printable.WriteString(`\0`)
		case '\t':
			printable.WriteString(`\t`)
		case '\n':
			printable.WriteString(`\n`)
		case '\r':
			printable.WriteString(`\r`)
		default:
			printable.WriteRune(c)
		}
	}

	var uni strings.Builder
	uni.WriteString(" (")
	for i, c := range grapheme {
		if i != 0 {
			uni.WriteByte(' ')
		}
		_, _ = fmt.Fprintf(&uni, "U+%04x", c)
	}
	uni.WriteByte(')')

	var dec string
	if len(grapheme) == 1 && grapheme[0] < 0x80 {
		dec = fmt.Sprintf(" Dec %d", grapheme[0])
	}

	var hex strings.Builder
	for i, c := range grapheme {
		if i != 0 {
			hex.WriteString(" +")
		}
		for _, by := range []byte(string(c)) {
			_, _ = fmt.Fprintf(&hex, " %02x", by)
		}
	}

	return fmt.Sprintf(`"%s"%s%s Hex%s`,
		printable.String(), uni.String(), dec, hex.String())
}
