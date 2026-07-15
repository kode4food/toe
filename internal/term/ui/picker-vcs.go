package ui

import (
	"path/filepath"

	"github.com/kode4food/toe/internal/view"
)

type changedFilePickerSource struct {
	PickerBase
}

const (
	changedFileModifiedIcon = "\uf044" //  - nf-fa-pencil_square_o
	changedFileAddedIcon    = "\uf4d0" //  - nf-oct-file_added
	changedFileRemovedIcon  = "\uf4d6" //  - nf-oct-file_removed
	changedFileRenamedIcon  = "\uf45a" //  - nf-oct-diff_renamed
)

// NewChangedFilePicker lists workspace files the version-control system
// reports as changed
func NewChangedFilePicker(e *view.Editor) *Picker {
	return NewPicker(e, &changedFilePickerSource{
		PickerBase: PickerBase{
			id:          "changed-files",
			columns:     []string{"", ""},
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
		Columns:     []string{changedFileIcon(fc.Kind), display},
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

func changedFileIcon(kind view.FileChangeKind) string {
	switch kind {
	case view.FileChangeUntracked:
		return "?"
	case view.FileChangeAdded:
		return changedFileAddedIcon
	case view.FileChangeConflict:
		return "!"
	case view.FileChangeDeleted:
		return changedFileRemovedIcon
	case view.FileChangeRenamed:
		return changedFileRenamedIcon
	default:
		return changedFileModifiedIcon
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
