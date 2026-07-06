package action

import (
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type posChar struct {
	pos int
	ch  rune
}

// SelectTextObjectAround selects around the text object identified by ch
// Plaintext objects: w=word, W=WORD, p=paragraph, m or bracket = surround pair
func SelectTextObjectAround(e *view.Editor, ch rune) {
	textObjectSelect(e, ch, core.TextObjectAround)
}

// SelectTextObjectInside selects inside the text object identified by ch
// Plaintext objects: w=word, W=WORD, p=paragraph, m or bracket = surround pair
func SelectTextObjectInside(e *view.Editor, ch rune) {
	textObjectSelect(e, ch, core.TextObjectInside)
}

// SurroundAdd wraps each selection with the pair that matches ch, then switches
// to normal mode. ch may be either the opening or closing bracket character
func SurroundAdd(e *view.Editor, ch rune) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	openCh, closeCh := core.GetPair(ch)
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	rawChanges := make([]core.Change, 0, len(ranges)*2)
	newRanges := make([]core.Range, len(ranges))
	offs := 0
	for i, r := range ranges {
		from, to := r.From(), r.To()
		rawChanges = append(rawChanges,
			core.TextChange(from+offs, from+offs, string(openCh)),
			core.TextChange(to+offs+1, to+offs+1, string(closeCh)),
		)
		newRanges[i] = core.NewRange(from+offs, to+offs+2)
		offs += 2
	}

	cs, err := core.NewChangeSetFromChanges(text, rawChanges)
	if err != nil {
		return
	}
	newSel, _ := core.NewSelection(newRanges, sel.PrimaryIndex())
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
	e.SetMode(view.ModeNormal)
}

// SurroundDelete removes the surrounding pair identified by ch around each
// selection, then switches to normal mode
func SurroundDelete(e *view.Editor, ch rune) {
	res, ok := resolveSurroundPos(e, ch)
	if !ok {
		return
	}
	slices.Sort(res.positions)
	rawChanges := make([]core.Change, len(res.positions))
	for i, p := range res.positions {
		rawChanges[i] = core.DeleteChange(p, p+1)
	}
	cs, err := core.NewChangeSetFromChanges(res.text, rawChanges)
	if err != nil {
		return
	}
	tx := core.NewTransaction(res.text).WithChanges(cs)
	_ = e.Apply(tx)
	e.SetMode(view.ModeNormal)
}

// SurroundReplace replaces the surrounding pair identified by from with the
// pair matching to. Called after two key prompts resolved by the model
func SurroundReplace(e *view.Editor, from, to rune) {
	res, ok := resolveSurroundPos(e, from)
	if !ok {
		return
	}

	openCh, closeCh := core.GetPair(to)

	sorted := make([]posChar, 0, len(res.positions))
	for i := 0; i < len(res.positions); i += 2 {
		sorted = append(sorted,
			posChar{res.positions[i], openCh},
			posChar{res.positions[i+1], closeCh},
		)
	}
	slices.SortFunc(sorted, func(a, b posChar) int {
		return a.pos - b.pos
	})
	rawChanges := make([]core.Change, len(sorted))
	for i, pc := range sorted {
		rawChanges[i] = core.TextChange(pc.pos, pc.pos+1, string(pc.ch))
	}
	cs, err := core.NewChangeSetFromChanges(res.text, rawChanges)
	if err != nil {
		return
	}
	tx := core.NewTransaction(res.text).WithChanges(cs)
	_ = e.Apply(tx)
	e.SetMode(view.ModeNormal)
}

func textObjectSelect(e *view.Editor, ch rune, kind core.TextObjectKind) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	n := max(countOrOne(e), 1)
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	newRanges := make([]core.Range, len(sel.Ranges()))
	for i, r := range sel.Ranges() {
		var nr core.Range
		switch ch {
		case 'w':
			nr = core.TextObjectWord(text, r, kind, false)
		case 'W':
			nr = core.TextObjectWord(text, r, kind, true)
		case 'p':
			nr = core.TextObjectParagraph(text, r, kind, n)
		case 'm':
			nr = r.TextObjectPairSurround(text, kind, 0, n)
		default:
			if !core.CharIsWord(ch) {
				nr = r.TextObjectPairSurround(text, kind, ch, n)
			} else {
				// Tree-sitter textobjects not yet supported; leave unchanged
				nr = r
			}
		}
		newRanges[i] = nr
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

type resolveSurroundPosRes struct {
	text      core.Rope
	sel       core.Selection
	positions []int
}

func resolveSurroundPos(e *view.Editor, ch rune) (resolveSurroundPosRes, bool) {
	v, ok := e.FocusedView()
	if !ok {
		return resolveSurroundPosRes{}, false
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return resolveSurroundPosRes{}, false
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	skip := max(countOrOne(e), 1)
	var positions []int
	var err error
	if ch != 'm' {
		positions, err = core.GetSurroundPosFor(text, sel, ch, skip)
	} else {
		positions, err = core.GetSurroundPos(text, sel, skip)
	}
	if err != nil {
		return resolveSurroundPosRes{}, false
	}
	return resolveSurroundPosRes{text, sel, positions}, true
}
