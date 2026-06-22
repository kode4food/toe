package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
	insertEntry struct {
		del  core.Deletion
		newR core.Range
		pair bool
	}

	rangeKind int
)

const (
	kindNormal rangeKind = iota
	kindAutoPair
	kindDup
)

const defaultYankRegister = '"'

// DeleteSelection deletes all selections and enters normal mode
func DeleteSelection(e *view.Editor) {
	DeleteSelectionNoyank(e)
}

// ChangeSelection deletes the selection and enters insert mode
// For linewise (whole-line) selections, opens a blank line above
func ChangeSelection(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	linewise := selectionIsLinewise(text, sel)

	ranges := sel.Ranges()
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = defaultYankRegister
	}

	// Yank first, then delete
	values := make([]string, 0, len(ranges))
	for _, r := range ranges {
		frag, err := r.MinWidth1(text).Slice(text)
		if err != nil {
			continue
		}
		values = append(values, frag.String())
	}
	e.Registers().Write(reg, values)

	if !applyDeletions(e, applyDeletionsArgs{text, sel, ranges}) {
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

// DeleteSelectionNoyank deletes each selection without yanking first
func DeleteSelectionNoyank(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	if !applyDeletions(e, applyDeletionsArgs{text, sel, sel.Ranges()}) {
		return
	}
	e.SetMode(view.ModeNormal)
}

// ChangeSelectionNoyank deletes each selection without yanking and enters
// insert mode
func ChangeSelectionNoyank(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	linewise := selectionIsLinewise(text, sel)
	if !applyDeletions(e, applyDeletionsArgs{text, sel, sel.Ranges()}) {
		return
	}
	if linewise {
		OpenAbove(e)
		return
	}
	e.SetMode(view.ModeInsert)
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
