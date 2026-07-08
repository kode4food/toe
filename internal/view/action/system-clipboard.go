package action

import "github.com/kode4food/toe/internal/view"

type SystemClipboard struct{}

var _ view.Clipboard = (*SystemClipboard)(nil)

func NewSystemClipboard() *SystemClipboard {
	return &SystemClipboard{}
}

func (*SystemClipboard) Available() bool {
	return clipboardAvailable()
}

func (*SystemClipboard) Write(text string) error {
	return writeClipboard(text)
}

func (*SystemClipboard) WritePrimary(text string) error {
	return writePrimaryClipboard(text)
}

func (*SystemClipboard) Read() (string, error) {
	return readClipboard()
}

func (*SystemClipboard) ReadPrimary() (string, error) {
	return readPrimaryClipboard()
}
