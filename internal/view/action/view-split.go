package action

import "github.com/kode4food/toe/internal/view"

// CloseCurrentView closes the focused view. If the document has unsaved
// changes and there are other views, the close is blocked
func CloseCurrentView(e *view.Editor) {
	doc := e.FocusedDocument()
	if doc != nil && doc.Modified() {
		if e.Tree().Count() > 1 {
			return
		}
	}
	e.CloseCurrentView()
}

// ResizeViewLeft pushes the focused split's border in the left direction by
// count cells (see [view.Editor.ResizeFocusedSplit])
func ResizeViewLeft(e *view.Editor) {
	resizeFocusedSplit(e, view.DirectionLeft)
}

// ResizeViewRight pushes the focused split's border in the right direction by
// count cells (see [view.Editor.ResizeFocusedSplit])
func ResizeViewRight(e *view.Editor) {
	resizeFocusedSplit(e, view.DirectionRight)
}

// ResizeViewUp pushes the focused split's border in the up direction by count
// cells (see [view.Editor.ResizeFocusedSplit])
func ResizeViewUp(e *view.Editor) {
	resizeFocusedSplit(e, view.DirectionUp)
}

// ResizeViewDown pushes the focused split's border in the down direction by
// count cells (see [view.Editor.ResizeFocusedSplit])
func ResizeViewDown(e *view.Editor) {
	resizeFocusedSplit(e, view.DirectionDown)
}

func resizeFocusedSplit(e *view.Editor, dir view.Direction) {
	delta := max(e.Count(), 1)
	e.ResetCount()
	e.ResizeFocusedSplit(dir, delta)
}
