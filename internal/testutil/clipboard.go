package testutil

import "github.com/kode4food/toe/internal/view"

// FakeClipboard is an in-memory view.Clipboard for hermetic tests, keeping the
// system and PRIMARY selections in separate buffers
type FakeClipboard struct {
	System  string
	Primary string
	Ready   bool
}

var _ view.Clipboard = (*FakeClipboard)(nil)

func NewFakeClipboard() *FakeClipboard {
	return &FakeClipboard{Ready: true}
}

func (c *FakeClipboard) Available() bool {
	return c.Ready
}

func (c *FakeClipboard) Write(text string) error {
	c.System = text
	return nil
}

func (c *FakeClipboard) WritePrimary(text string) error {
	c.Primary = text
	return nil
}

func (c *FakeClipboard) Read() (string, error) {
	return c.System, nil
}

func (c *FakeClipboard) ReadPrimary() (string, error) {
	return c.Primary, nil
}
