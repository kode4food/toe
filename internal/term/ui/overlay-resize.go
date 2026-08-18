package ui

import "github.com/kode4food/toe/internal/geom"

// a border drag that rescales a centered overlay, measured as a delta from the
// scales in effect when the drag began, so the grabbed border tracks the
// pointer whatever the overlay is anchored to
type overlayDrag struct {
	from        geom.Point
	startWidth  float64
	startHeight float64
	signX       int
	signY       int
}

const (
	minOverlayScale = 0.3
	maxOverlayScale = 1.0
)

// arms the drag only when the point lands on a border
func (d *overlayDrag) begin(bounds geom.Area, at geom.Point) bool {
	if !bounds.Contains(at) {
		return false
	}
	switch at.X {
	case bounds.X:
		d.signX = -1
	case bounds.Right():
		d.signX = 1
	}
	switch at.Y {
	case bounds.Y:
		d.signY = -1
	case bounds.Bottom():
		d.signY = 1
	}
	d.from = at
	return d.active()
}

func (d *overlayDrag) active() bool {
	return d.signX != 0 || d.signY != 0
}

// an axis whose border was not grabbed keeps its saved scale
func (d *overlayDrag) applyTo(
	opts PickerLayoutOptions, id string, at geom.Point, avail geom.Size,
) PickerLayoutOptions {
	if d.signX != 0 {
		opts = opts.withWidthScale(id, d.widthScale(at, avail.Width))
	}
	if d.signY != 0 {
		opts = opts.withHeightScale(id, d.heightScale(at, avail.Height))
	}
	return opts
}

func (d *overlayDrag) widthScale(at geom.Point, avail int) float64 {
	return scaleDelta(d.startWidth, d.signX*(at.X-d.from.X), avail)
}

func (d *overlayDrag) heightScale(at geom.Point, avail int) float64 {
	return scaleDelta(d.startHeight, d.signY*(at.Y-d.from.Y), avail)
}

// a centered overlay grows on both sides at once, so a border that moves n
// cells changes the size by 2n
func scaleDelta(start float64, moved, avail int) float64 {
	if avail <= 0 {
		return clampOverlayScale(start)
	}
	return clampOverlayScale(start + float64(2*moved)/float64(avail))
}

func scaleExtent(avail int, scale float64) int {
	return max(int(float64(avail)*scale+0.5), 0)
}

func clampOverlayScale(scale float64) float64 {
	return min(max(scale, minOverlayScale), maxOverlayScale)
}
