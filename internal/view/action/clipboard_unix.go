//go:build !windows && !darwin

package action

import "os/exec"

func clipboardAvailable() bool {
	for _, t := range []string{"xclip", "xsel", "wl-copy"} {
		if _, err := exec.LookPath(t); err == nil {
			return true
		}
	}
	return false
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
