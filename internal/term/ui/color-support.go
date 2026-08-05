package ui

import (
	"os"
	"strconv"
	"strings"
)

// an empty value matches any non-empty value for env
type trueColorTerm struct {
	env   string
	value string
}

// vte 0.36 is the first release with 24-bit color support
const minTrueColorVTE = 3600

// add new terminals here as their support is confirmed
var trueColorTerms = []trueColorTerm{
	{env: "COLORTERM", value: "truecolor"},
	{env: "COLORTERM", value: "24bit"},
	{env: "WSL_DISTRO_NAME"},
	{env: "KITTY_WINDOW_ID"},
	{env: "WEZTERM_EXECUTABLE"},
	{env: "ALACRITTY_WINDOW_ID"},
	{env: "KONSOLE_VERSION"},
	{env: "WT_SESSION"}, // windows terminal
	{env: "TERM_PROGRAM", value: "ghostty"},
	{env: "TERM_PROGRAM", value: "iTerm.app"},
	{env: "TERM_PROGRAM", value: "WezTerm"},
	{env: "TERM_PROGRAM", value: "rio"},
	{env: "TERM_PROGRAM", value: "vscode"},
	{env: "TERM", value: "xterm-kitty"},
	{env: "TERM", value: "xterm-ghostty"},
	{env: "TERM", value: "alacritty"},
	{env: "TERM", value: "contour"},
	{env: "TERM", value: "foot"},
	{env: "TERM", value: "foot-extra"},
	{env: "TERM", value: "rio"},
	{env: "TERM", value: "wezterm"},
}

// TrueColorSupported reports whether the terminal can render 24-bit color,
// based on well-known environment variables
func TrueColorSupported() bool {
	if trueColorVTE() {
		return true
	}
	// terminfo names its 24-bit entries with a -direct suffix
	term := os.Getenv("TERM")
	if strings.HasSuffix(term, "-direct") ||
		strings.Contains(term, "truecolor") {
		return true
	}
	for _, t := range trueColorTerms {
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

func trueColorVTE() bool {
	v, err := strconv.Atoi(os.Getenv("VTE_VERSION"))
	return err == nil && v >= minTrueColorVTE
}
