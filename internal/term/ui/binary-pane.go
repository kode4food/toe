package ui

import (
	"io"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
)

// BinaryPane displays a read-only, responsive hexadecimal dump
type BinaryPane struct {
	id     view.Id
	editor *view.Editor
	area   geom.Area
	dirty  bool
	path   string
	size   int64
	offset int64
}

const binarySessionOffsetKey = "offset"

var (
	_ view.Pane  = (*BinaryPane)(nil)
	_ PaneInput  = (*BinaryPane)(nil)
	_ PaneCursor = (*BinaryPane)(nil)
)

// NewBinaryPane opens path as a read-only binary dump
func NewBinaryPane(e *view.Editor, path string) (*BinaryPane, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	return &BinaryPane{
		editor: e,
		path:   abs,
		size:   info.Size(),
		dirty:  true,
	}, nil
}

// HandleEvent scrolls the binary dump
func (p *BinaryPane) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, bool) {
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case m.Code == tea.KeyDown || m.Text == "j":
			p.scrollRows(1)
		case m.Code == tea.KeyUp || m.Text == "k":
			p.scrollRows(-1)
		case m.Code == tea.KeyPgDown || m.Code == tea.KeyKpPgDown:
			p.scrollRows(p.contentRows())
		case m.Code == tea.KeyPgUp || m.Code == tea.KeyKpPgUp:
			p.scrollRows(-p.contentRows())
		case m.Code == tea.KeyHome:
			p.setOffset(0)
		case m.Code == tea.KeyEnd || m.Text == "G":
			p.setOffset(p.maxOffset())
		default:
			return ignored(), false
		}
	case tea.MouseWheelMsg:
		n := max(cx.Editor.Options().ScrollLines, 1)
		switch m.Button {
		case tea.MouseWheelUp:
			p.scrollRows(-n)
		case tea.MouseWheelDown:
			p.scrollRows(n)
		default:
			return ignored(), false
		}
	default:
		return ignored(), false
	}
	return consumed(), true
}

// Cursor reports that a binary pane has no text cursor
func (p *BinaryPane) Cursor(*Context) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

// ID returns the pane identifier
func (p *BinaryPane) ID() view.Id {
	return p.id
}

// SetID sets the pane identifier
func (p *BinaryPane) SetID(id view.Id) {
	p.id = id
}

// Area returns the screen rectangle assigned by the layout tree
func (p *BinaryPane) Area() geom.Area {
	return p.area
}

// SetArea sets the screen rectangle assigned by the layout tree
func (p *BinaryPane) SetArea(a geom.Area) {
	if a == p.area {
		return
	}
	p.area = a
	p.setOffset(p.offset)
	p.dirty = true
}

// MarkDirty flags the pane as needing a repaint
func (p *BinaryPane) MarkDirty() {
	p.dirty = true
}

// ConsumeDirty reports and clears whether the pane changed
func (p *BinaryPane) ConsumeDirty() bool {
	dirty := p.dirty
	p.dirty = false
	return dirty
}

// Mode reports binary dump mode
func (p *BinaryPane) Mode() view.Mode {
	return view.ModeBinary
}

// Path returns the binary file path
func (p *BinaryPane) Path() string {
	return p.path
}

// SaveSession stores the binary path and viewport offset
func (p *BinaryPane) SaveSession(w *view.SessionWriter) {
	w.SaveSlot(view.SessionKindBinary, p.path)
	w.SaveValue(binarySessionOffsetKey, p.offset)
}

// Split returns another pane displaying the same binary dump
func (p *BinaryPane) Split() (view.Pane, error) {
	return &BinaryPane{
		editor: p.editor,
		path:   p.path,
		size:   p.size,
		offset: p.offset,
		dirty:  true,
	}, nil
}

// Close closes this binary pane
func (p *BinaryPane) Close() {
	if p.editor != nil {
		p.editor.RemovePane(p.id)
	}
}

// Discard releases this displaced binary pane
func (p *BinaryPane) Discard() {
}

// Shutdown releases external resources owned by this pane
func (p *BinaryPane) Shutdown() {
}

func (p *BinaryPane) readVisible() ([]byte, error) {
	n := p.bytesPerRow() * p.contentRows()
	return readBinaryRange(p.path, p.offset, n)
}

func (p *BinaryPane) scrollRows(rows int) {
	p.setOffset(p.offset + int64(rows*p.bytesPerRow()))
}

func (p *BinaryPane) setOffset(offset int64) {
	row := int64(p.bytesPerRow())
	offset = min(max(offset, 0), p.maxOffset())
	offset -= offset % row
	if offset != p.offset {
		p.offset = offset
		p.dirty = true
	}
}

func (p *BinaryPane) maxOffset() int64 {
	rows := (p.size + int64(p.bytesPerRow()) - 1) /
		int64(p.bytesPerRow())
	return max(rows-int64(p.contentRows()), 0) *
		int64(p.bytesPerRow())
}

func (p *BinaryPane) bytesPerRow() int {
	return binaryBytesPerRow(binaryBytesPerRowArgs{
		width:       p.area.Width,
		offsetWidth: p.offsetWidth(),
	})
}

func (p *BinaryPane) contentRows() int {
	return max(p.area.Height-1, 1)
}

func (p *BinaryPane) offsetWidth() int {
	return binaryOffsetWidth(p.size)
}

func registerBinaryPane(e *view.Editor) {
	e.RegisterPaneRestorer(view.SessionKindBinary,
		func(e *view.Editor, session *view.PaneSession) (view.Pane, error) {
			pane, err := NewBinaryPane(e, session.Path())
			if err != nil {
				return nil, err
			}
			if offset, ok := session.Value(binarySessionOffsetKey); ok {
				switch value := offset.(type) {
				case int:
					pane.offset = int64(value)
				case int64:
					pane.offset = value
				}
				pane.setOffset(pane.offset)
			}
			return pane, nil
		})
}

func readBinaryRange(path string, offset int64, count int) ([]byte, error) {
	if count <= 0 {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data := make([]byte, count)
	n, err := f.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return data[:n], nil
}
