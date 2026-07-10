package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type insertEntry struct {
	del  core.Deletion
	newR core.Range
	pair bool
}

// DeleteSelection yanks selections into the active register, then deletes them
func DeleteSelection(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok || doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	yankSelectionRanges(e, text, ranges)
	if !applyDeletions(e, applyDeletionsArgs{
		text: text, sel: sel, ranges: ranges,
	}) {
		return
	}
	e.SetMode(view.ModeNormal)
}

// ChangeSelection yanks all selections into the active register, deletes them,
// and enters insert mode. For linewise selections, opens a blank line above
func ChangeSelection(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok || doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	linewise := selectionIsLinewise(text, sel)
	ranges := sel.Ranges()
	yankSelectionRanges(e, text, ranges)
	if !applyDeletions(e, applyDeletionsArgs{
		text: text, sel: sel, ranges: ranges,
	}) {
		return
	}
	if linewise {
		OpenAbove(e)
		return
	}
	e.SetMode(view.ModeInsert)
}

// SplitSelectionOnNewline splits each selection range on line boundaries,
// producing one sub-range per line (excluding the line ending itself)
func SplitSelectionOnNewline(e *view.Editor) {
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

	var newRanges []core.Range
	for _, r := range sel.Ranges() {
		if r.From() == r.To() {
			newRanges = append(newRanges, r)
			continue
		}
		from := r.From()
		to := r.To()
		pos := from
		for pos < to {
			line, err := text.CharToLine(pos)
			if err != nil {
				break
			}
			lineEnd, err := text.LineEndCharIndex(line)
			if err != nil {
				break
			}
			lineTotal, err := text.LineToChar(line + 1)
			if err != nil {
				// Last line (no newline at end)
				end := min(to, text.LenChars())
				if pos < end {
					newRanges = append(newRanges, core.NewRange(pos, end))
				}
				break
			}
			// lineEnd = newline char pos; lineTotal = start of next line
			end := min(lineEnd, to)
			if pos < end {
				newRanges = append(newRanges, core.NewRange(pos, end))
			}
			pos = lineTotal
		}
	}

	if len(newRanges) == 0 {
		return
	}
	newSel, err := core.NewSelection(newRanges, 0)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// DeleteSelectionNoYank deletes each selection without yanking first
func DeleteSelectionNoYank(e *view.Editor) {
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
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	if !applyDeletions(e, applyDeletionsArgs{
		text: text, sel: sel, ranges: sel.Ranges(),
	}) {
		return
	}
	e.SetMode(view.ModeNormal)
}

// ChangeSelectionNoYank deletes each selection without yanking and enters
// insert mode
func ChangeSelectionNoYank(e *view.Editor) {
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
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	linewise := selectionIsLinewise(text, sel)
	if !applyDeletions(e, applyDeletionsArgs{
		text: text, sel: sel, ranges: sel.Ranges(),
	}) {
		return
	}
	if linewise {
		OpenAbove(e)
		return
	}
	e.SetMode(view.ModeInsert)
}

func yankSelectionRanges(e *view.Editor, text core.Rope, ranges []core.Range) {
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = view.RegisterDefaultYank
	}
	values := make([]string, 0, len(ranges))
	for _, r := range ranges {
		frag, err := r.MinWidth1(text).Slice(text)
		if err != nil {
			continue
		}
		values = append(values, frag.String())
	}
	e.WriteRegister(reg, values)
}

type applyChangesFromArgs struct {
	text    core.Rope
	sel     core.Selection
	ranges  []core.Range
	changes []core.Change
}

func applyChangesFrom(e *view.Editor, args applyChangesFromArgs) {
	if len(args.changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(args.text, args.changes)
	if err != nil {
		return
	}
	newRanges := make([]core.Range, len(args.ranges))
	for i, r := range args.ranges {
		mapped, err := cs.MapRange(r)
		if err != nil {
			return
		}
		newRanges[i] = core.PointRange(mapped.From())
	}
	newSel, err := core.NewSelection(newRanges, args.sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
	e.SetMode(view.ModeNormal)
}
