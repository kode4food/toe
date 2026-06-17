package core

type (
	// Position represents a zero-indexed point in a text buffer
	Position struct {
		Row int
		Col int
	}

	// VisualOffsetError describes why a visual position could not be resolved
	VisualOffsetError int
)

func (p Position) Add(q Position) Position {
	return Position{
		Row: p.Row + q.Row,
		Col: p.Col + q.Col,
	}
}

func (p Position) Sub(q Position) Position {
	return Position{
		Row: p.Row - q.Row,
		Col: p.Col - q.Col,
	}
}

func (p Position) IsZero() bool {
	return p.Row == 0 && p.Col == 0
}

func (p Position) Traverse(text string) Position {
	row := p.Row
	col := p.Col
	for _, ch := range text {
		if CharIsLineEnding(ch) {
			row++
			col = 0
			continue
		}
		col++
	}
	return Position{Row: row, Col: col}
}
