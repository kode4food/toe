//go:build windows

package action

const ttyDevice = "CONOUT$"

func clipboardAvailable() bool {
	return false
}

func writeClipboard(_ string) error {
	return ErrNoClipboardProvider
}

func writePrimaryClipboard(_ string) error {
	return ErrNoClipboardProvider
}

func readClipboard() (string, error) {
	return "", ErrNoClipboardProvider
}

func readPrimaryClipboard() (string, error) {
	return "", ErrNoClipboardProvider
}
