package geom

type (
	// Size is a pixel or cell extent, named so a call site never has to guess
	// the order a bare (int, int) would leave ambiguous
	Size struct {
		Width, Height int
	}

	// Point is a screen or document coordinate in cells
	Point struct {
		X, Y int
	}

	// Area is a screen rectangle: an origin Point and a Size
	Area struct {
		Point
		Size
	}
)

// Contains reports whether a Point lies inside the specified Size
func (s Size) Contains(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < s.Width && p.Y < s.Height
}

// Contains reports whether a Point lies inside the specified Area
func (a Area) Contains(p Point) bool {
	return p.X >= a.X && p.X < a.X+a.Width &&
		p.Y >= a.Y && p.Y < a.Y+a.Height
}

// Translate returns a copy of Area with its origin shifted by Point
func (a Area) Translate(d Point) Area {
	a.X += d.X
	a.Y += d.Y
	return a
}
