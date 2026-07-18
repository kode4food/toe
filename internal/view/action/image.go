package action

import "github.com/kode4food/toe/internal/view"

type imageZoomer interface {
	ZoomIn()
	ZoomOut()
	ResetZoom()
}

// ImageZoomIn increases the focused image's scale
func ImageZoomIn(e *view.Editor) {
	if p, ok := e.FocusedPane().(imageZoomer); ok {
		p.ZoomIn()
	}
}

// ImageZoomOut decreases the focused image's scale
func ImageZoomOut(e *view.Editor) {
	if p, ok := e.FocusedPane().(imageZoomer); ok {
		p.ZoomOut()
	}
}

// ImageZoomReset restores the focused image's fitted scale
func ImageZoomReset(e *view.Editor) {
	if p, ok := e.FocusedPane().(imageZoomer); ok {
		p.ResetZoom()
	}
}
