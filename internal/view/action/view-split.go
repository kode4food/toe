package action

import "github.com/kode4food/toe/internal/view"

// CloseCurrentView closes the focused view. If the document has unsaved changes and there are other
// views, the close is blocked
func CloseCurrentView(e *view.Editor) {
	doc := e.FocusedDocument()
	if doc != nil && doc.Modified() {
		if e.Tree().Count() > 1 {
			return
		}
	}
	e.CloseCurrentView()
}

// ResizeViewLeft pushes the focused split's border left by count cells
func ResizeViewLeft(e *view.Editor) {
	resizeFocusedSplit(e, view.DirectionLeft)
}

// ResizeViewRight pushes the focused split's border right by count cells
func ResizeViewRight(e *view.Editor) {
	resizeFocusedSplit(e, view.DirectionRight)
}

// ResizeViewUp pushes the focused split's border up by count cells
func ResizeViewUp(e *view.Editor) {
	resizeFocusedSplit(e, view.DirectionUp)
}

// ResizeViewDown pushes the focused split's border down by count cells
func ResizeViewDown(e *view.Editor) {
	resizeFocusedSplit(e, view.DirectionDown)
}

func resizeFocusedSplit(e *view.Editor, dir view.Direction) {
	e.ResizeFocusedSplit(dir, e.CountOr(1))
}
