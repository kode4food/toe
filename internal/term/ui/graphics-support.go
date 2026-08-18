package ui

// graphicsTerms lists terminals known to support Kitty graphics. Add new
// terminals here as their support is confirmed
var graphicsTerms = []termMatch{
	{env: "KITTY_WINDOW_ID"},              // kitty
	{env: "TERM", value: "xterm-kitty"},   // kitty
	{env: "TERM", value: "xterm-ghostty"}, // ghostty
	{env: "TERM_PROGRAM", value: "ghostty"},
}

func graphicsSupported() bool {
	return matchesTerm(graphicsTerms)
}
