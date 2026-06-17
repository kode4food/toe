package tui

type (
	StyledGrapheme struct {
		Symbol string
		Style  Style
	}

	Span struct {
		Content string
		Style   Style
	}

	Spans []Span
)
