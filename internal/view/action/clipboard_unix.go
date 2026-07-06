//go:build !windows && !darwin

package action

import "os/exec"

// ShowClipboardProvider returns the name of the detected clipboard provider.
// ttyAvail reports whether OSC 52 output is possible
func ShowClipboardProvider(ttyAvail func() bool) string {
	tools := []string{"xclip", "xsel", "wl-copy"}
	if ttyAvail != nil && ttyAvail() {
		for _, t := range tools {
			if _, err := exec.LookPath(t); err == nil {
				return "osc52+" + t
			}
		}
		return "osc52"
	}
	for _, t := range tools {
		if _, err := exec.LookPath(t); err == nil {
			return t
		}
	}
	return "none"
}

func writeClipboard(text string) error {
	if tryWriteCmds([][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"wl-copy"},
	}, text) {
		return nil
	}
	return ErrNoClipboardProvider
}

func writePrimaryClipboard(text string) error {
	if tryWriteCmds([][]string{
		{"xclip", "-selection", "primary"},
		{"xsel", "--primary", "--input"},
		{"wl-copy", "--primary"},
	}, text) {
		return nil
	}
	return ErrNoClipboardProvider
}

func readClipboard() (string, error) {
	if v, ok := tryReadCmds([][]string{
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
		{"wl-paste", "--no-newline"},
	}); ok {
		return v, nil
	}
	return "", ErrNoClipboardProvider
}

func readPrimaryClipboard() (string, error) {
	if v, ok := tryReadCmds([][]string{
		{"xclip", "-selection", "primary", "-o"},
		{"xsel", "--primary", "--output"},
		{"wl-paste", "--primary", "--no-newline"},
	}); ok {
		return v, nil
	}
	return readClipboard()
}
