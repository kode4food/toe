package ui

import (
	"path/filepath"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

type changedFilePickerSource struct {
	PickerBase
}

const (
	fileModifiedIcon  = "\uf459" //  - nf-oct-diff_modified
	fileAddedIcon     = "\uf457" //  - nf-oct-diff_added
	fileRemovedIcon   = "\uf458" //  - nf-oct-diff_removed
	fileRenamedIcon   = "\uf45a" //  - nf-oct-diff_renamed
	fileUntrackedIcon = "\uf420" //  - nf-oct-question
	fileConflictIcon  = "\uf421" //  - nf-oct-alert
)

var (
	changedFileIcons = [...]string{
		view.FileChangeUntracked: fileUntrackedIcon,
		view.FileChangeAdded:     fileAddedIcon,
		view.FileChangeModified:  fileModifiedIcon,
		view.FileChangeConflict:  fileConflictIcon,
		view.FileChangeDeleted:   fileRemovedIcon,
		view.FileChangeRenamed:   fileRenamedIcon,
	}

	changedFileScopes = [...]string{
		view.FileChangeUntracked: "diff.plus",
		view.FileChangeAdded:     "diff.plus",
		view.FileChangeModified:  "diff.delta",
		view.FileChangeConflict:  "error",
		view.FileChangeDeleted:   "diff.minus",
		view.FileChangeRenamed:   "diff.delta.moved",
	}
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
		e.SetStatusMsg(i18n.Text(
			i18n.ErrorMessage, i18n.Vars{"message": err},
		))
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
	v, ok := AcceptPath(e, item.Location.Target.Path, action)
	if !ok {
		return
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return
	}
	if sel, ok := lineRangeSelection(doc.Text(), item.Location.Lines); ok {
		doc.SetSelectionFor(v.ID(), sel)
	}
	AlignAcceptedView(e, v, doc)
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
	if kind < 0 || int(kind) >= len(changedFileIcons) {
		return fileModifiedIcon
	}
	return changedFileIcons[kind]
}

func changedFileScope(kind view.FileChangeKind) string {
	if kind < 0 || int(kind) >= len(changedFileScopes) {
		return ""
	}
	return changedFileScopes[kind]
}

func lineRangeSelection(
	text core.Rope, lr *PickerLineRange,
) (core.Selection, bool) {
	if lr == nil || lr.From < 0 {
		return core.Selection{}, false
	}
	lineStart, err := text.LineToChar(lr.From)
	if err != nil {
		return core.Selection{}, false
	}
	lineEnd := text.LenChars()
	if end, err := text.LineToChar(lr.From + 1); err == nil {
		lineEnd = end
	}
	sel, err := core.NewSelection(
		[]core.Range{core.NewRange(lineStart, lineEnd)}, 0,
	)
	if err != nil {
		return core.Selection{}, false
	}
	return sel, true
}
