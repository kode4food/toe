package action

import "github.com/kode4food/toe/internal/view"

// CloseCurrentView closes the focused view. If the document has unsaved
// changes and there are other views, the close is blocked
func CloseCurrentView(e *view.Editor) {
	doc, _ := e.FocusedDocument()
	if doc != nil && doc.Modified() {
		if e.Tree().Count() > 1 {
			return
		}
	}
	e.CloseCurrentView()
}

// CloseCurrentViewForce closes the focused view unconditionally
func CloseCurrentViewForce(e *view.Editor) {
	e.CloseCurrentView()
}

// HSplit opens the current pane in a new horizontal split (stacked)
func HSplit(e *view.Editor) {
	if err := e.SplitFocused(view.LayoutHorizontal); err != nil {
		e.SetStatusMsg(err.Error())
	}
}

// VSplit opens the current pane in a new vertical split (side by side)
func VSplit(e *view.Editor) {
	if err := e.SplitFocused(view.LayoutVertical); err != nil {
		e.SetStatusMsg(err.Error())
	}
}

// TransposeView flips the layout of the split container holding the focused
// view
func TransposeView(e *view.Editor) {
	e.Transpose()
}

// JumpViewLeft moves focus to the nearest split to the left
func JumpViewLeft(e *view.Editor) {
	e.FocusDirection(view.DirectionLeft)
}

// JumpViewRight moves focus to the nearest split to the right
func JumpViewRight(e *view.Editor) {
	e.FocusDirection(view.DirectionRight)
}

// JumpViewUp moves focus to the nearest split above
func JumpViewUp(e *view.Editor) {
	e.FocusDirection(view.DirectionUp)
}

// JumpViewDown moves focus to the nearest split below
func JumpViewDown(e *view.Editor) {
	e.FocusDirection(view.DirectionDown)
}

// SwapViewLeft swaps the focused split with the one to its left
func SwapViewLeft(e *view.Editor) {
	e.SwapSplitInDirection(view.DirectionLeft)
}

// SwapViewRight swaps the focused split with the one to its right
func SwapViewRight(e *view.Editor) {
	e.SwapSplitInDirection(view.DirectionRight)
}

// SwapViewUp swaps the focused split with the one above it
func SwapViewUp(e *view.Editor) {
	e.SwapSplitInDirection(view.DirectionUp)
}

// SwapViewDown swaps the focused split with the one below it
func SwapViewDown(e *view.Editor) {
	e.SwapSplitInDirection(view.DirectionDown)
}

// RotateView cycles focus to the next view in tree order, wrapping around
func RotateView(e *view.Editor) {
	e.FocusNextView()
}

// CloseOtherViews closes every view except the currently focused one
func CloseOtherViews(e *view.Editor) {
	e.CloseAllOtherViews()
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
