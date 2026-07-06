//go:build darwin

package action

import "os/exec"

// ShowClipboardProvider returns the name of the detected clipboard provider.
// ttyAvail reports whether OSC 52 output is possible
func ShowClipboardProvider(ttyAvail func() bool) string {
	if ttyAvail != nil && ttyAvail() {
		return "osc52+pbcopy"
	}
	if _, err := exec.LookPath("pbcopy"); err == nil {
		return "pbcopy"
	}
	return "none"
}

func writeClipboard(text string) error {
	if tryWriteCmds([][]string{{"pbcopy"}}, text) {
		return nil
	}
	return ErrNoClipboardProvider
}

func writePrimaryClipboard(text string) error {
	return writeClipboard(text)
}

func readClipboard() (string, error) {
	if v, ok := tryReadCmds([][]string{{"pbpaste"}}); ok {
		return v, nil
	}
	return "", ErrNoClipboardProvider
}

func readPrimaryClipboard() (string, error) {
	return readClipboard()
}
