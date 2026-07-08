package action

import "github.com/kode4food/toe/internal/view"

type osc52Clipboard struct {
	view.Clipboard
	tty TTYWriter
}

// NewOSC52Clipboard layers OSC 52 terminal writes over inner, so a copy also
// reaches the clipboard of a terminal reached over ssh
func NewOSC52Clipboard(inner view.Clipboard) view.Clipboard {
	return osc52Clipboard{Clipboard: inner, tty: MakeTTYWriter()}
}

func (c osc52Clipboard) Write(text string) error {
	c.tty(text, false)
	return c.Clipboard.Write(text)
}

func (c osc52Clipboard) WritePrimary(text string) error {
	c.tty(text, true)
	return c.Clipboard.WritePrimary(text)
}
