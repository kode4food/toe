package action

import (
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

// Yank copies the text of every selection range to the active register
// (defaulting to '"') and exits select mode
func Yank(e *view.Editor) {
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
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = view.RegisterDefaultYank
	}
	values := yankFragments(text, sel)
	e.WriteRegister(reg, values)
	setYankStatus(e, reg, len(values))
	e.SetMode(view.ModeNormal)
}

// PasteAfter pastes the active register's contents after each selection
func PasteAfter(e *view.Editor) {
	pasteImpl(e, false)
	e.SetMode(view.ModeNormal)
}

// PasteBefore pastes the active register's contents before each selection
func PasteBefore(e *view.Editor) {
	pasteImpl(e, true)
	e.SetMode(view.ModeNormal)
}

// ReplaceWithYanked replaces each selection with the corresponding value from
// the active register (default '"'). Exits select mode
func ReplaceWithYanked(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = view.RegisterDefaultYank
	}
	values := e.ReadRegister(reg)
	if len(values) == 0 {
		return
	}
	n := max(e.Count(), 1)
	valueFor := func(i int) string {
		v := values[len(values)-1]
		if i < len(values) {
			v = values[i]
		}
		return strings.Repeat(v, n)
	}

	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	// valueFor uses a sequential counter that advances only for non-empty
	// ranges
	valueIdx := 0
	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		if r.Empty() {
			continue
		}
		changes = append(changes,
			core.TextChange(r.From(), r.To(), valueFor(valueIdx)),
		)
		valueIdx++
	}
	applyChangesFrom(e, applyChangesFromArgs{
		text: text, sel: sel, ranges: ranges, changes: changes,
	})
}

func yankFragments(text core.Rope, sel core.Selection) []string {
	parts := make([]string, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		parts = append(parts, frag)
	}
	return parts
}

func pasteImpl(e *view.Editor, before bool) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = view.RegisterDefaultYank
	}
	values := e.ReadRegister(reg)
	if len(values) == 0 {
		return
	}

	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	linewise := false
	for _, val := range values {
		if len(val) > 0 && val[len(val)-1] == '\n' {
			linewise = true
			break
		}
	}

	valueFor := func(i int) string {
		if i < len(values) {
			return values[i]
		}
		return values[len(values)-1]
	}

	pastePos := make([]int, len(ranges))
	for i := range pastePos {
		pastePos[i] = -1
	}
	changes := make([]core.Change, 0, len(ranges))
	for i, r := range ranges {
		pos, ok := pastePosition(text, r, linewise, before)
		if !ok {
			continue
		}
		pastePos[i] = pos
		changes = append(changes, core.TextChange(pos, pos, valueFor(i)))
	}
	if len(changes) == 0 {
		return
	}

	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}

	newRanges := make([]core.Range, len(ranges))
	for i, r := range ranges {
		if pastePos[i] < 0 {
			newRanges[i] = r
			continue
		}
		newPos, err := cs.MapPos(pastePos[i], core.AssocBeforeSticky)
		if err != nil {
			newRanges[i] = r
			continue
		}
		newRanges[i] = core.PointRange(newPos)
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

func setYankStatus(e *view.Editor, reg rune, n int) {
	if n == 0 {
		return
	}
	key := i18n.StatusYankedSelection
	if n != 1 {
		key = i18n.StatusYankedSelections
	}
	e.SetStatusMsg(i18n.Text(key, i18n.Vars{
		"count":    n,
		"register": string(reg),
	}))
}

func pastePosition(
	text core.Rope, r core.Range, linewise, before bool,
) (int, bool) {
	if !linewise {
		if before {
			return r.From(), true
		}
		return r.To(), true
	}
	if before {
		line, err := text.CharToLine(r.From())
		if err != nil {
			return 0, false
		}
		pos, err := text.LineToChar(line)
		if err != nil {
			return 0, false
		}
		return pos, true
	}
	line, err := text.CharToLine(r.To())
	if err != nil {
		return 0, false
	}
	next := line + 1
	if next >= text.LenLines() {
		return text.LenChars(), true
	}
	pos, err := text.LineToChar(next)
	if err != nil {
		return 0, false
	}
	return pos, true
}
