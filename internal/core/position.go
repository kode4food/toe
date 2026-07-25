package core

// Position is a 1-based line and column document address, the way a person
// names a location. Distinct from geom.Point, which is a 0-based screen cell
type Position struct {
	Line int
	Col  int
}

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
	return Position{Line: line + 1, Col: char - start + 1}, nil
}
