package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type insertEntry struct {
	del      core.Span
	newRange core.Range
	pair     bool
}

// DeleteSelection yanks selections into the active register, then deletes them
func DeleteSelection(e *view.Editor) {
	deleteOrChange(e, deleteOrChangeArgs{yank: true})
}

// ChangeSelection yanks all selections into the active register, deletes them,
// and enters insert mode. For linewise selections, opens a blank line above
func ChangeSelection(e *view.Editor) {
	deleteOrChange(e, deleteOrChangeArgs{yank: true, enterInsert: true})
}

// SplitSelectionOnNewline splits each selection range on line boundaries,
// producing one sub-range per line (excluding the line ending itself)
func SplitSelectionOnNewline(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
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
					newRanges = append(newRanges, core.Range{
						Anchor: pos,
						Head:   end,
					})
				}
				break
			}
			// lineEnd = newline char pos, lineTotal = start of next line
			end := min(lineEnd, to)
			if pos < end {
				newRanges = append(newRanges, core.Range{
					Anchor: pos,
					Head:   end,
				})
			}
			pos = lineTotal
		}
	}

	if len(newRanges) == 0 {
		return
	}
	if newSel, err := core.NewSelection(newRanges, 0); err == nil {
		doc.SetSelectionFor(v.ID(), newSel)
	}
}

// DeleteSelectionNoYank deletes each selection without yanking first
func DeleteSelectionNoYank(e *view.Editor) {
	deleteOrChange(e, deleteOrChangeArgs{})
}

// ChangeSelectionNoYank deletes each selection without yanking and enters
// insert mode
func ChangeSelectionNoYank(e *view.Editor) {
	deleteOrChange(e, deleteOrChangeArgs{enterInsert: true})
}

type deleteOrChangeArgs struct {
	yank        bool
	enterInsert bool
}

func deleteOrChange(e *view.Editor, args deleteOrChangeArgs) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil || doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	linewise := args.enterInsert && selectionIsLinewise(text, sel)
	ranges := sel.Ranges()
	if args.yank {
		yankSelectionRanges(e, text, ranges)
	}
	if !applyDeletions(e, applyDeletionsArgs{
		text:   text,
		sel:    sel,
		ranges: ranges,
	}) {
		return
	}
	switch {
	case linewise:
		openImpl(openArgs{editor: e, count: 1})
	case args.enterInsert:
		e.SetMode(view.ModeInsert)
	default:
		e.SetMode(view.ModeNormal)
	}
}

func yankSelectionRanges(e *view.Editor, text core.Rope, ranges []core.Range) {
	reg := e.DeleteRegister()
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
		if mapped, err := cs.MapRange(r); err == nil {
			newRanges[i] = core.PointRange(mapped.From())
			continue
		}
		return
	}
	newSel, err := core.NewSelection(newRanges, args.sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
	e.SetMode(view.ModeNormal)
}
