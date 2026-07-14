package ui

import (
	"path/filepath"

	"github.com/kode4food/toe/internal/view"
)

type changedFilePickerSource struct {
	PickerBase
}

// NewChangedFilePicker lists workspace files the version-control system
// reports as changed
func NewChangedFilePicker(e *view.Editor) *Picker {
	return NewPicker(e, &changedFilePickerSource{
		PickerBase: PickerBase{
			id:          "changed-files",
			columns:     []string{"change", "path"},
			matchColumn: 1,
			proportions: []int{0, 1},
		},
	})
}

func (c *changedFilePickerSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	vc := e.VersionControl()
	if vc == nil {
		return nil, nil, func() {}
	}
	changes, err := vc.ChangedFiles()
	if err != nil {
		e.SetStatusMsg("error: " + err.Error())
		return nil, nil, func() {}
	}
	// providers report symlink-resolved paths; resolve the workspace root
	// the same way so names relativize cleanly
	cwd := e.Cwd()
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}

	feed := make(chan PickerItem)
	done := make(chan struct{})
	go func() {
		defer close(feed)
		for _, fc := range changes {
			item := changedFileItem(vc, fc, cwd)
			select {
			case feed <- item:
			case <-done:
				return
			}
		}
	}()
	stop := func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}
	return nil, feed, stop
}

func (c *changedFilePickerSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	_, _ = AcceptPath(e, item.Location.Target.Path, action)
}

func changedFileItem(
	vc view.VersionControl, fc view.FileChange, cwd string,
) PickerItem {
	display := view.DocumentRelativeName(fc.Path, cwd)
	if fc.Kind == view.FileChangeRenamed {
		display = view.DocumentRelativeName(fc.FromPath, cwd) +
			" → " + display
	}
	hunks := changedFileHunks(vc, fc)
	return PickerItem{
		Display:     display,
		Columns:     []string{changedFileLabel(fc.Kind), display},
		StyleScopes: []string{changedFileScope(fc.Kind), ""},
		SortKey:     display,
		DiffHunks:   hunks,
		Location: PickerLocation{
			Target: PickerTarget{Path: fc.Path},
			Lines:  firstChangeLines(hunks),
		},
	}
}

func changedFileHunks(
	vc view.VersionControl, fc view.FileChange,
) []view.DiffHunk {
	switch fc.Kind {
	case view.FileChangeUntracked, view.FileChangeAdded,
		view.FileChangeDeleted:
		// the whole file is new or gone; there is no base to diff against
		return nil
	default:
		return vc.DiffHunksForPath(fc.Path)
	}
}

func firstChangeLines(hunks []view.DiffHunk) *PickerLineRange {
	if len(hunks) == 0 {
		return nil
	}
	h := hunks[0]
	return &PickerLineRange{From: h.From, To: max(h.From, h.To-1)}
}

func changedFileLabel(kind view.FileChangeKind) string {
	switch kind {
	case view.FileChangeUntracked:
		return "untracked"
	case view.FileChangeAdded:
		return "added"
	case view.FileChangeConflict:
		return "conflict"
	case view.FileChangeDeleted:
		return "deleted"
	case view.FileChangeRenamed:
		return "renamed"
	default:
		return "modified"
	}
}

func changedFileScope(kind view.FileChangeKind) string {
	switch kind {
	case view.FileChangeUntracked, view.FileChangeAdded:
		return "diff.plus"
	case view.FileChangeModified:
		return "diff.delta"
	case view.FileChangeConflict:
		return "error"
	case view.FileChangeDeleted:
		return "diff.minus"
	case view.FileChangeRenamed:
		return "diff.delta.moved"
	default:
		return ""
	}
}
