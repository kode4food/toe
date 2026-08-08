package core

type (
	// Position is a 1-based line and column document address, the way a person
	// names a location. Distinct from geom.Point, a screen cell
	Position struct {
		Line   int
		Column int
	}

	// LinePos addresses a point inside a line by 0-based line index and the
	// absolute character offset of the point, not a column
	LinePos struct {
		Line int
		Pos  int
	}
)

// Position returns the 1-based line and column of a character offset
func (r Rope) Position(char int) (Position, error) {
	line, err := r.CharToLine(char)
	if err != nil {
		return Position{}, err
	}
	start, err := r.LineToChar(line)
	if err != nil {
		return Position{}, err
	}
	return Position{Line: line + 1, Column: char - start + 1}, nil
}
