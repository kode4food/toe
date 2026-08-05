package action

import "github.com/kode4food/toe/internal/view"

// SystemClipboard talks to the OS clipboard via an external command, detected
// once at construction and reused for every read and write
type SystemClipboard struct {
	provider clipboardProvider
}

var _ view.Clipboard = (*SystemClipboard)(nil)

// NewSystemClipboard returns the host's clipboard provider
func NewSystemClipboard() *SystemClipboard {
	return &SystemClipboard{provider: detectClipboardProvider()}
}

// Available reports whether the host offers a clipboard
func (c *SystemClipboard) Available() bool {
	return c.provider.read != nil
}

// Write copies text to the system clipboard
func (c *SystemClipboard) Write(text string) error {
	return runWrite(c.provider.write, text)
}

// WritePrimary copies text to the primary selection
func (c *SystemClipboard) WritePrimary(text string) error {
	if c.provider.write == nil {
		return ErrNoClipboardProvider
	}
	if c.provider.writePrim == nil {
		return nil
	}
	return runWrite(c.provider.writePrim, text)
}

// Read returns the system clipboard contents
func (c *SystemClipboard) Read() (string, error) {
	return runRead(c.provider.read)
}

// ReadPrimary returns the primary selection contents
func (c *SystemClipboard) ReadPrimary() (string, error) {
	if c.provider.read == nil {
		return "", ErrNoClipboardProvider
	}
	if c.provider.readPrim == nil {
		return "", nil
	}
	return runRead(c.provider.readPrim)
}
