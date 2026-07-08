package action

import (
	"golang.design/x/clipboard"

	"github.com/kode4food/toe/internal/view"
)

type SystemClipboard struct {
	ready bool
}

var _ view.Clipboard = (*SystemClipboard)(nil)

func NewSystemClipboard() *SystemClipboard {
	return &SystemClipboard{
		ready: clipboard.Init() == nil,
	}
}

func (c *SystemClipboard) Available() bool {
	return c.ready
}

func (c *SystemClipboard) Write(text string) error {
	if !c.ready {
		return ErrNoClipboardProvider
	}
	clipboard.Write(clipboard.FmtText, []byte(text))
	return nil
}

func (c *SystemClipboard) WritePrimary(string) error {
	return nil
}

func (c *SystemClipboard) Read() (string, error) {
	if !c.ready {
		return "", ErrNoClipboardProvider
	}
	return string(clipboard.Read(clipboard.FmtText)), nil
}

func (c *SystemClipboard) ReadPrimary() (string, error) {
	return c.Read()
}
