package view

type (
	// Clipboard is the system clipboard plus the X11 PRIMARY selection
	Clipboard interface {
		Read() (string, error)
		ReadPrimary() (string, error)
		Write(text string) error
		WritePrimary(text string) error
		Available() bool
	}

	noopClipboard struct{}
)

func (noopClipboard) Read() (string, error) {
	return "", nil
}

func (noopClipboard) ReadPrimary() (string, error) {
	return "", nil
}

func (noopClipboard) Write(string) error {
	return nil
}

func (noopClipboard) WritePrimary(string) error {
	return nil
}

func (noopClipboard) Available() bool {
	return false
}
