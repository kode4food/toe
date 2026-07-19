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

// Empty reports whether Size has no usable cells
func (s Size) Empty() bool {
	return s.Width <= 0 || s.Height <= 0
}

// Add returns p offset by d
func (p Point) Add(d Point) Point {
	return Point{X: p.X + d.X, Y: p.Y + d.Y}
}

// Sub returns p offset by the negation of d
func (p Point) Sub(d Point) Point {
	return Point{X: p.X - d.X, Y: p.Y - d.Y}
}

// Contains reports whether a Point lies inside the specified Size
func (s Size) Contains(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < s.Width && p.Y < s.Height
}

// Clamp returns p pinned to the last valid cell of the Size, i.e. into the
// range [0, Width-1] x [0, Height-1]
func (s Size) Clamp(p Point) Point {
	return Point{
		X: min(max(p.X, 0), max(s.Width-1, 0)),
		Y: min(max(p.Y, 0), max(s.Height-1, 0)),
	}
}

// Contains reports whether a Point lies inside the specified Area
func (a Area) Contains(p Point) bool {
	return p.X >= a.X && p.X < a.X+a.Width &&
		p.Y >= a.Y && p.Y < a.Y+a.Height
}

// Right is the x of the rightmost cell (inclusive): X + Width - 1
func (a Area) Right() int {
	return a.X + a.Width - 1
}

// Bottom is the y of the bottommost cell (inclusive): Y + Height - 1
func (a Area) Bottom() int {
	return a.Y + a.Height - 1
}

// Translate returns a copy of Area with its origin shifted by Point
func (a Area) Translate(d Point) Area {
	a.Point = a.Point.Add(d)
	return a
}

// Intersects reports whether a and b overlap in at least one cell
func (a Area) Intersects(b Area) bool {
	if a.Empty() || b.Empty() {
		return false
	}
	return a.X < b.X+b.Width && b.X < a.X+a.Width &&
		a.Y < b.Y+b.Height && b.Y < a.Y+a.Height
}

// Center returns the origin Point that centers inner within a, clamped so the
// origin never precedes a's own origin when inner is larger
func (a Area) Center(inner Size) Point {
	return Point{
		X: a.X + max(a.Width-inner.Width, 0)/2,
		Y: a.Y + max(a.Height-inner.Height, 0)/2,
	}
}

// Inset returns a shrunk toward its center by by.Width on the left and right
// and by.Height on the top and bottom, with the Size clamped to zero
func (a Area) Inset(by Size) Area {
	return Area{
		Point: Point{X: a.X + by.Width, Y: a.Y + by.Height},
		Size: Size{
			Width:  max(a.Width-2*by.Width, 0),
			Height: max(a.Height-2*by.Height, 0),
		},
	}
}
