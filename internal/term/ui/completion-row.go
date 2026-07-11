package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type completionRowParts struct {
	icon  string
	label string
	info  string
}

const completionPreviewMaxWidth = 40

func (c *completionComponent) width() int {
	w := completionMinWidth
	for _, item := range c.items {
		w = max(w, c.rowWidth(item, true)+2)
	}
	if len(c.items) > completionMaxRows {
		w += completionScrollGap
	}
	return w + 2
}

func (c *completionComponent) rowWidth(
	item view.CompletionItem, selected bool,
) int {
	return runewidth.StringWidth(c.rowLeft(item, selected))
}

type renderCompletionRowArgs struct {
	item     view.CompletionItem
	selected bool
	query    string
	base     tui.Style
	match    tui.Style
	icon     tui.Style
	info     tui.Style
}

func (c *completionComponent) renderRow(
	buf *tui.Buffer, x, y, w, listW int, args renderCompletionRowArgs,
) {
	buf.SetString(x, y, clipPad("", w), args.base)
	parts := c.rowParts(args.item, args.selected)
	labelX := x
	budget := listW
	if parts.icon != "" {
		next := writeCompletionPart(
			buf, labelX, y, budget, parts.icon, args.icon,
		)
		budget -= next - labelX
		labelX = next
		if budget > 0 {
			buf.SetString(labelX, y, " ", args.base)
			labelX++
			budget--
		}
	}
	if budget <= 0 {
		return
	}
	writePickerMatched(buf, writePickerMatchedArgs{
		x: labelX, y: y, maxW: budget, text: parts.label,
		indices: completionLabelMatchIndices(parts.label, args.query),
		base:    args.base, match: args.match,
	})
	used := min(runewidth.StringWidth(parts.label), budget)
	labelX += used
	budget -= used
	if parts.info == "" || budget <= 1 {
		return
	}
	buf.SetString(labelX, y, " ", args.base)
	labelX++
	budget--
	writeCompletionPart(buf, labelX, y, budget, parts.info, args.info)
}

func (c *completionComponent) rowLeft(
	item view.CompletionItem, selected bool,
) string {
	return completionRowText(c.rowParts(item, selected))
}

func (c *completionComponent) rowParts(
	item view.CompletionItem, selected bool,
) completionRowParts {
	return completionRowPartsFor(item, c.opts.Icons, selected)
}

func (c *completionComponent) renderScroll(
	buf *tui.Buffer, x, y, rows int, style tui.Style,
) {
	if rows <= 0 || len(c.items) <= rows {
		return
	}
	scrollH := min((rows*rows+len(c.items)-1)/len(c.items), rows)
	scrollY := 0
	if len(c.items) > rows {
		scrollY = (rows - scrollH) * c.scroll / (len(c.items) - rows)
	}
	for i := range scrollH {
		buf.SetString(x, y+scrollY+i, "▌", style)
	}
}

func completionRowPartsFor(
	item view.CompletionItem, icons CompletionIconMode, selected bool,
) completionRowParts {
	parts := completionRowParts{
		icon:  completionKindMarker(item.Kind, icons),
		label: item.Label,
	}
	if selected {
		parts.label += strings.Join(strings.Fields(item.LabelDetail), " ")
		var info []string
		if detail := completionRowDetail(item); detail != "" {
			info = append(info, detail)
		}
		desc := strings.Join(strings.Fields(item.LabelDescription), " ")
		if desc != "" {
			info = append(info, desc)
		}
		if item.Deprecated {
			info = append(info, "deprecated")
		}
		parts.info = completionPreview(strings.Join(info, " "))
	}
	return parts
}

func completionRowText(parts completionRowParts) string {
	out := parts.label
	if parts.icon != "" {
		out = parts.icon + " " + out
	}
	if parts.info != "" {
		out += " " + parts.info
	}
	return out
}

func completionRowDetail(item view.CompletionItem) string {
	detail := strings.Join(strings.Fields(item.Detail), " ")
	labelDetail := strings.Join(strings.Fields(item.LabelDetail), " ")
	if detail == "" || detail == labelDetail {
		return ""
	}
	return detail
}

func completionPreview(s string) string {
	if s == "" {
		return ""
	}
	return runewidth.Truncate(s, completionPreviewMaxWidth, "...")
}

func writeCompletionPart(
	buf *tui.Buffer, x, y, maxW int, text string, st tui.Style,
) int {
	if maxW <= 0 || text == "" {
		return x
	}
	text = runewidth.Truncate(text, maxW, "")
	buf.SetString(x, y, text, st)
	return x + runewidth.StringWidth(text)
}

func completionLabelMatchIndices(label, query string) []int {
	if query == "" {
		return nil
	}
	rs := []rune(label)
	if strings.HasPrefix(strings.ToLower(label), strings.ToLower(query)) {
		n := min(utf8.RuneCountInString(query), len(rs))
		indices := make([]int, n)
		for i := range n {
			indices[i] = i
		}
		return indices
	}
	indices := make([]int, 0, utf8.RuneCountInString(query))
	from := 0
	for _, q := range query {
		q = unicode.ToLower(q)
		found := -1
		for i := from; i < len(rs); i++ {
			if unicode.ToLower(rs[i]) == q {
				found = i
				break
			}
		}
		if found < 0 {
			return nil
		}
		indices = append(indices, found)
		from = found + 1
	}
	return indices
}
