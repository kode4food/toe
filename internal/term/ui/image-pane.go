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
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
)

type (
	// ImagePane displays an image in the editor's pane tree, zooming and
	// panning it in response to keys and mouse, and shows no text cursor
	ImagePane struct {
		viewport viewportState
		wheel    wheelState

		id     view.Id
		editor *view.Editor
		area   geom.Area
		dirty  bool
		path   string
		image  *Image
	}

	viewportState struct {
		zoom int
		pan  geom.Point
	}

	wheelState struct {
		at      time.Time
		zooming bool
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
	imageWheelZoomStep = 10
	minImageZoom       = 25
	maxImageZoom       = 400
	imageKeyPanStep    = 4
	// wheelGestureGap is the longest pause that still counts as one continuous
	// scroll, covering the cadence of momentum events after a key is released
	wheelGestureGap = 200 * time.Millisecond

	imageSessionZoomKey = "zoom"
	imageSessionPanXKey = "pan_x"
	imageSessionPanYKey = "pan_y"

	// wheelMods: holding any of these swaps the bare-wheel action between pan
	// and zoom, the closest a terminal gets to pinch-to-zoom
	wheelMods = tea.ModCtrl | tea.ModAlt | tea.ModSuper
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
		editor: e, path: abs, image: img, dirty: true,
		viewport: viewportState{zoom: defaultImageZoom},
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

// HandleEvent handles image mouse input: a click zooms in, a modified click
// zooms out, a bare wheel pans, and a modified wheel zooms. Everything else
// falls through to the editor
func (p *ImagePane) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, bool) {
	switch m := msg.(type) {
	case tea.MouseClickMsg:
		if m.Button != tea.MouseLeft {
			return ignored(), false
		}
		if m.Mod&wheelMods != 0 {
			p.ZoomOut()
		} else {
			p.ZoomIn()
		}
		return consumed(), true
	case tea.MouseWheelMsg:
		zooming := m.Mod&wheelMods != 0
		// hold a zoom gesture through the modifier-less momentum tail, but let
		// a pan gesture switch to zoom the moment the modifier appears
		if !zooming && p.wheel.zooming &&
			time.Since(p.wheel.at) < wheelGestureGap {
			zooming = true
		}
		p.wheel.at, p.wheel.zooming = time.Now(), zooming
		// drop input while the last change is still transmitting, so a fast
		// burst is paced to the pipeline instead of queuing a laggy tail
		id := kittyImageID(p.image.ContentID(), uint32(p.id), false)
		if cx.images.inFlight(id) {
			return consumed(), true
		}
		if zooming {
			return p.wheelZoom(m.Button)
		}
		return p.wheelPan(m.Button)
	default:
		return ignored(), false
	}
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
	w.SaveValue(imageSessionZoomKey, p.viewport.zoom)
	w.SaveValue(imageSessionPanXKey, p.viewport.pan.X)
	w.SaveValue(imageSessionPanYKey, p.viewport.pan.Y)
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
	return p.viewport.zoom
}

// ZoomIn increases the image scale
func (p *ImagePane) ZoomIn() {
	p.setZoom(p.viewport.zoom + imageKeyZoomStep)
}

// ZoomOut decreases the image scale
func (p *ImagePane) ZoomOut() {
	p.setZoom(p.viewport.zoom - imageKeyZoomStep)
}

// ResetZoom restores the fitted image scale and recenters the view
func (p *ImagePane) ResetZoom() {
	p.setZoom(defaultImageZoom)
	p.setPan(geom.Point{})
}

// Pan returns the view offset from center, in grid cells
func (p *ImagePane) Pan() geom.Point {
	return p.viewport.pan
}

// PanBy shifts the view by the given cell delta
func (p *ImagePane) PanBy(delta geom.Point) {
	p.setPan(geom.Point{
		X: p.viewport.pan.X + delta.X,
		Y: p.viewport.pan.Y + delta.Y,
	})
}

func (p *ImagePane) PanLeft()  { p.PanBy(geom.Point{X: -imageKeyPanStep}) }
func (p *ImagePane) PanRight() { p.PanBy(geom.Point{X: imageKeyPanStep}) }
func (p *ImagePane) PanUp()    { p.PanBy(geom.Point{Y: -imageKeyPanStep}) }
func (p *ImagePane) PanDown()  { p.PanBy(geom.Point{Y: imageKeyPanStep}) }

// Split returns another pane displaying the same image
func (p *ImagePane) Split() (view.Pane, error) {
	return &ImagePane{
		editor:   p.editor,
		path:     p.path,
		image:    p.image,
		dirty:    true,
		viewport: p.viewport,
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

func (p *ImagePane) wheelZoom(button tea.MouseButton) (EventResult, bool) {
	switch button {
	case tea.MouseWheelUp:
		p.setZoom(p.viewport.zoom + imageWheelZoomStep)
	case tea.MouseWheelDown:
		p.setZoom(p.viewport.zoom - imageWheelZoomStep)
	default:
		return ignored(), false
	}
	return consumed(), true
}

func (p *ImagePane) wheelPan(button tea.MouseButton) (EventResult, bool) {
	switch button {
	case tea.MouseWheelUp:
		p.PanUp()
	case tea.MouseWheelDown:
		p.PanDown()
	case tea.MouseWheelLeft:
		p.PanLeft()
	case tea.MouseWheelRight:
		p.PanRight()
	default:
		return ignored(), false
	}
	return consumed(), true
}

func (p *ImagePane) setPan(pan geom.Point) {
	bound := p.panBound()
	pan.X = min(max(pan.X, -bound.X), bound.X)
	pan.Y = min(max(pan.Y, -bound.Y), bound.Y)
	if pan != p.viewport.pan {
		p.viewport.pan = pan
		p.dirty = true
	}
}

func (p *ImagePane) setZoom(zoom int) {
	zoom = min(max(zoom, minImageZoom), maxImageZoom)
	if zoom != p.viewport.zoom {
		p.viewport.zoom = zoom
		p.setPan(p.viewport.pan)
		p.dirty = true
	}
}

func (p *ImagePane) panBound() geom.Point {
	contentH := max(p.area.Height-1, 0)
	cells := imagePaneCellSize(imagePaneCellSizeArgs{
		pane:     p,
		maxCells: geom.Size{Width: p.area.Width, Height: contentH},
		pixels:   p.image.Size(),
	})
	visW := min(cells.Width, p.area.Width)
	visH := min(cells.Height, contentH)
	return geom.Point{
		X: (cells.Width - visW + 1) / 2,
		Y: (cells.Height - visH + 1) / 2,
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

// restorePan sets the saved offset directly, not via setPan: panMax is zero
// until the first render, which would clamp it to center; render reclamps later
func (p *ImagePane) restorePan(session *view.PaneSession) {
	x, okX := sessionInt(session, imageSessionPanXKey)
	y, okY := sessionInt(session, imageSessionPanYKey)
	if !okX && !okY {
		return
	}
	p.viewport.pan = geom.Point{X: x, Y: y}
	p.dirty = true
}

// sessionInt reads an int session value, tolerating the int64 that a
// round-tripped session yields
func sessionInt(session *view.PaneSession, key string) (int, bool) {
	value, ok := session.Value(key)
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	}
	return 0, false
}

func registerImagePane(e *view.Editor) {
	e.RegisterPaneRestorer(view.SessionKindImage,
		func(e *view.Editor, session *view.PaneSession) (view.Pane, error) {
			pane, err := NewImagePane(e, session.Path())
			if err != nil {
				return nil, err
			}
			pane.restoreZoom(session)
			pane.restorePan(session)
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
