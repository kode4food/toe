package ui

import (
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
)

type (
	// ImagePane displays an image in the editor's pane tree. It owns its own
	// input: mouse wheel zooms via MouseHandler, and it shows no text cursor
	ImagePane struct {
		id     view.Id
		editor *view.Editor
		area   geom.Area
		dirty  bool
		path   string
		image  *Image
		zoom   int
	}

	// Image holds decoded image data, its content identifier, and the decoded
	// source format (e.g. "png") for transmission fast paths
	Image struct {
		image.Image
		id     uint32
		format string
	}
)

const (
	defaultImageZoom   = 100
	imageKeyZoomStep   = 25
	imageWheelZoomStep = 5
	minImageZoom       = 25
	maxImageZoom       = 400

	imageSessionZoomKey = "zoom"
)

// ErrInvalidImage reports a file that cannot be decoded as an image
var ErrInvalidImage = errors.New("invalid image")

var (
	_ view.Pane  = (*ImagePane)(nil)
	_ PaneInput  = (*ImagePane)(nil)
	_ PaneCursor = (*ImagePane)(nil)
)

// LoadImage reads path and returns a decoded image
func LoadImage(path string) (*Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidImage, path)
	}
	hash := fnv.New32a()
	_, _ = hash.Write(data)
	return &Image{Image: img, id: hash.Sum32(), format: format}, nil
}

// NewImagePane loads path into an image pane
func NewImagePane(e *view.Editor, path string) (*ImagePane, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	img, err := LoadImage(abs)
	if err != nil {
		return nil, err
	}
	return &ImagePane{
		editor: e, path: abs, image: img, zoom: defaultImageZoom,
		dirty: true,
	}, nil
}

// Size returns the image bounds in pixels
func (i *Image) Size() geom.Size {
	b := i.Bounds()
	return geom.Size{Width: b.Dx(), Height: b.Dy()}
}

// ContentID returns a stable identifier for the decoded image bytes
func (i *Image) ContentID() uint32 {
	return i.id
}

// HandleEvent zooms the image on a wheel event and ignores everything else,
// letting keys and other input fall through to the editor
func (p *ImagePane) HandleEvent(
	_ *Context, msg tea.Msg,
) (EventResult, bool) {
	wheel, ok := msg.(tea.MouseWheelMsg)
	if !ok {
		return ignored(), false
	}
	switch wheel.Button {
	case tea.MouseWheelUp:
		p.setZoom(p.zoom + imageWheelZoomStep)
	case tea.MouseWheelDown:
		p.setZoom(p.zoom - imageWheelZoomStep)
	default:
		return ignored(), false
	}
	return consumed(), true
}

// Cursor reports that an image pane shows no text cursor
func (p *ImagePane) Cursor(*Context) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

// ID returns the pane identifier
func (p *ImagePane) ID() view.Id {
	return p.id
}

// SetID sets the pane identifier
func (p *ImagePane) SetID(id view.Id) {
	p.id = id
}

// Area returns the screen rectangle assigned by the layout tree
func (p *ImagePane) Area() geom.Area {
	return p.area
}

// SetArea sets the screen rectangle assigned by the layout tree
func (p *ImagePane) SetArea(a geom.Area) {
	if a != p.area {
		p.area = a
		p.dirty = true
	}
}

// MarkDirty flags the pane as needing a repaint
func (p *ImagePane) MarkDirty() {
	p.dirty = true
}

// ConsumeDirty reports and clears whether the pane changed
func (p *ImagePane) ConsumeDirty() bool {
	dirty := p.dirty
	p.dirty = false
	return dirty
}

// Mode reports image mode
func (p *ImagePane) Mode() view.Mode {
	return view.ModeImage
}

// Path returns the loaded image path
func (p *ImagePane) Path() string {
	return p.path
}

// SaveSession stores the image path so the pane can be reopened
func (p *ImagePane) SaveSession(w *view.SessionWriter) {
	w.SaveSlot(view.SessionKindImage, p.path)
	w.SaveValue(imageSessionZoomKey, p.zoom)
}

// Image returns the decoded image
func (p *ImagePane) Image() *Image {
	return p.image
}

// Reload re-decodes the backing file after an external change; the new bytes
// yield a new ContentID, so the display path retransmits automatically
func (p *ImagePane) Reload() error {
	img, err := LoadImage(p.path)
	if err != nil {
		return err
	}
	p.image = img
	p.dirty = true
	return nil
}

// Zoom returns the image scale as a percentage of its fitted size
func (p *ImagePane) Zoom() int {
	return p.zoom
}

// ZoomIn increases the image scale
func (p *ImagePane) ZoomIn() {
	p.setZoom(p.zoom + imageKeyZoomStep)
}

// ZoomOut decreases the image scale
func (p *ImagePane) ZoomOut() {
	p.setZoom(p.zoom - imageKeyZoomStep)
}

// ResetZoom restores the fitted image scale
func (p *ImagePane) ResetZoom() {
	p.setZoom(defaultImageZoom)
}

// Split returns another pane displaying the same image
func (p *ImagePane) Split() (view.Pane, error) {
	return &ImagePane{
		editor: p.editor, path: p.path, image: p.image, dirty: true,
		zoom: p.zoom,
	}, nil
}

// Close closes this image pane
func (p *ImagePane) Close() {
	if p.editor != nil {
		p.editor.RemovePane(p.id)
	}
}

// Discard releases this displaced image pane
func (p *ImagePane) Discard() {
}

// Shutdown releases external resources owned by this pane
func (p *ImagePane) Shutdown() {
}

func (p *ImagePane) setZoom(zoom int) {
	zoom = min(max(zoom, minImageZoom), maxImageZoom)
	if zoom != p.zoom {
		p.zoom = zoom
		p.dirty = true
	}
}

func (p *ImagePane) restoreZoom(session *view.PaneSession) {
	value, ok := session.Value(imageSessionZoomKey)
	if !ok {
		return
	}
	switch zoom := value.(type) {
	case int:
		p.setZoom(zoom)
	case int64:
		zoom = min(max(zoom, int64(minImageZoom)), int64(maxImageZoom))
		p.setZoom(int(zoom))
	}
}

func registerImagePane(e *view.Editor) {
	e.RegisterPaneRestorer(view.SessionKindImage,
		func(e *view.Editor, session *view.PaneSession) (view.Pane, error) {
			pane, err := NewImagePane(e, session.Path())
			if err != nil {
				return nil, err
			}
			pane.restoreZoom(session)
			return pane, nil
		})
}

// rangeImagePanes calls fn for each image pane in the editor's pane tree
func rangeImagePanes(e *view.Editor, fn func(*ImagePane)) {
	e.Tree().Range(func(p view.Pane) bool {
		if img, ok := p.(*ImagePane); ok {
			fn(img)
		}
		return true
	})
}
