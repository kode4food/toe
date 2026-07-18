package ui

import "os"

// graphicsTerm matches a terminal known to support the Kitty graphics
// protocol. An empty value matches any non-empty value for env
type graphicsTerm struct {
	env   string
	value string
}

// graphicsTerms lists terminals known to support Kitty graphics.
// Add new terminals here as their support is confirmed.
// ponytail: env sniff, not a runtime handshake; add a Kitty graphics
// query (\e_Gi=…,a=q…\e\\) if this misfires under tmux/ssh passthrough
var graphicsTerms = []graphicsTerm{
	{env: "KITTY_WINDOW_ID"},              // kitty
	{env: "TERM", value: "xterm-kitty"},   // kitty
	{env: "TERM", value: "xterm-ghostty"}, // ghostty
	{env: "TERM_PROGRAM", value: "ghostty"},
	{env: "TERM_PROGRAM", value: "WezTerm"},
	{env: "KONSOLE_VERSION"}, // konsole
}

// graphicsSupported reports whether the terminal can display images via the
// Kitty graphics protocol, based on well-known environment variables
func graphicsSupported() bool {
	for _, t := range graphicsTerms {
		v := os.Getenv(t.env)
		if v == "" {
			continue
		}
		if t.value == "" || t.value == v {
			return true
		}
	}
	return false
}
