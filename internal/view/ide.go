package view

// IDEServer is the MCP/IDE integration the editor exposes to AI coding CLIs.
// It can be turned off and back on, and reports the port it serves on
type IDEServer interface {
	Enabled() bool
	SetEnabled(enabled bool)
	Port() int
}

// SetIDEServer installs the IDE integration controller
func (e *Editor) SetIDEServer(s IDEServer) { e.ide = s }

// IDEServer returns the IDE integration controller, nil when none is installed
func (e *Editor) IDEServer() IDEServer { return e.ide }
