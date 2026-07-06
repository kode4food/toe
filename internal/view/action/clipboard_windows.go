//go:build windows

package action

const ttyDevice = "CONOUT$"

// ShowClipboardProvider returns the name of the detected clipboard provider.
// ttyAvail reports whether OSC 52 output is possible
func ShowClipboardProvider(ttyAvail func() bool) string {
	if ttyAvail != nil && ttyAvail() {
		return "osc52"
	}
	return "none"
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
