package action

import "github.com/kode4food/toe/internal/view"

// SystemClipboard talks to the OS clipboard via an external command, detected
// once at construction and reused for every read and write
type SystemClipboard struct {
	provider clipboardProvider
}

var _ view.Clipboard = (*SystemClipboard)(nil)

func NewSystemClipboard() *SystemClipboard {
	return &SystemClipboard{provider: detectClipboardProvider()}
}

func (c *SystemClipboard) Available() bool {
	return c.provider.read != nil
}

func (c *SystemClipboard) Write(text string) error {
	return runWrite(c.provider.write, text)
}

func (c *SystemClipboard) WritePrimary(text string) error {
	if c.provider.writePrim == nil {
		return c.Write(text)
	}
	return runWrite(c.provider.writePrim, text)
}

func (c *SystemClipboard) Read() (string, error) {
	return runRead(c.provider.read)
}

func (c *SystemClipboard) ReadPrimary() (string, error) {
	if c.provider.readPrim == nil {
		return c.Read()
	}
	return runRead(c.provider.readPrim)
}
