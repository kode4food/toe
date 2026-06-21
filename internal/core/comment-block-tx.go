package core

// CreateBlockCommentTransaction builds the transaction and updated ranges for a
// block comment toggle operation
func CreateBlockCommentTransaction(
	doc Rope, commented bool, commentChanges []CommentChange,
) (Transaction, []Range, error) {
	changes := make([]Change, 0, len(commentChanges)*2)
	var ranges []Range
	offs := 0

	for _, ch := range commentChanges {
		from := ch.Range.From()
		if commented {
			if ch.Kind != CommentChangeCommented {
				continue
			}
			startLen := len([]rune(ch.StartToken))
			endLen := len([]rune(ch.EndToken))
			sm := 0
			if ch.StartMargin {
				sm = 1
			}
			em := 0
			if ch.EndMargin {
				em = 1
			}
			changes = append(changes,
				DeleteChange(
					from+ch.StartPos,
					from+ch.StartPos+startLen+sm,
				),
				DeleteChange(
					from+ch.EndPos-endLen-em+1,
					from+ch.EndPos+1,
				),
			)
		} else {
			switch ch.Kind {
			case CommentChangeUncommented:
				startLen := len([]rune(ch.StartToken))
				endLen := len([]rune(ch.EndToken))
				sp := from + ch.StartPos
				ep := from + ch.EndPos
				changes = append(changes,
					TextChange(sp, sp, ch.StartToken+" "),
					TextChange(ep+1, ep+1, " "+ch.EndToken),
				)
				offset := startLen + endLen + 2
				rng := NewRange(
					from+offs, from+offs+ch.EndPos+1+offset,
				).WithDirection(ch.Range.Direction())
				ranges = append(ranges, rng)
				offs += offset
			default:
				f, t := ch.Range.From()+offs, ch.Range.To()+offs
				ranges = append(ranges, NewRange(f, t))
			}
		}
	}

	cs, err := NewChangeSetFromChanges(doc, changes)
	if err != nil {
		return Transaction{}, nil, err
	}
	return NewTransaction(doc).WithChanges(cs), ranges, nil
}

// ToggleBlockComments toggles block comments on all ranges in sel
func ToggleBlockComments(
	doc Rope, sel Selection, tokens []BlockCommentToken,
) (Transaction, error) {
	commented, commentChanges, err := FindBlockComments(tokens, doc, sel)
	if err != nil {
		return Transaction{}, err
	}
	tx, ranges, err := CreateBlockCommentTransaction(
		doc, commented, commentChanges,
	)
	if err != nil {
		return Transaction{}, err
	}
	if !commented && len(ranges) > 0 {
		pri := min(sel.PrimaryIndex(), len(ranges)-1)
		newSel, err := NewSelection(ranges, pri)
		if err != nil {
			return Transaction{}, err
		}
		tx = tx.WithSelection(newSel)
	}
	return tx, nil
}
