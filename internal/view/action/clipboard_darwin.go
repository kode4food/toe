//go:build darwin

package action

import "os/exec"

func clipboardAvailable() bool {
	_, err := exec.LookPath("pbcopy")
	return err == nil
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
