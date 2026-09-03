package ui

import (
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type changedFilePickerSource struct {
	PickerBase
}

const (
	fileModifiedIcon  = "\uf459" // '' - nf-oct-diff_modified
	fileAddedIcon     = "\uf457" // '' - nf-oct-diff_added
	fileRemovedIcon   = "\uf458" // '' - nf-oct-diff_removed
	fileRenamedIcon   = "\uf45a" // '' - nf-oct-diff_renamed
	fileUntrackedIcon = "\uf420" // '' - nf-oct-question
	fileConflictIcon  = "\uf421" // '' - nf-oct-alert
)

const (
	statusPickerStagedKey   i18n.Key = "status.pickerStaged"
	statusPickerUnstagedKey i18n.Key = "status.pickerUnstaged"
	statusPickerDiscardKey  i18n.Key = "status.pickerDiscard"
)

// git reports the index first, so staged rows lead the list
const (
	changedFileStaged = iota
	changedFileUnstaged
)

var (
	changedFileScopes = [...]string{
		view.FileChangeUntracked: "diff.plus",
		view.FileChangeAdded:     "diff.plus",
		view.FileChangeModified:  "diff.delta",
		view.FileChangeConflict:  "error",
		view.FileChangeDeleted:   "diff.minus",
		view.FileChangeRenamed:   "diff.delta.moved",
	}

	changedFileIcons = [...]string{
		view.FileChangeUntracked: fileUntrackedIcon,
		view.FileChangeAdded:     fileAddedIcon,
		view.FileChangeModified:  fileModifiedIcon,
		view.FileChangeConflict:  fileConflictIcon,
		view.FileChangeDeleted:   fileRemovedIcon,
		view.FileChangeRenamed:   fileRenamedIcon,
	}

	changedFileAscii = [...]string{
		view.FileChangeUntracked: "?",
		view.FileChangeAdded:     "A",
		view.FileChangeModified:  "M",
		view.FileChangeConflict:  "!",
		view.FileChangeDeleted:   "D",
		view.FileChangeRenamed:   "R",
	}
)

// NewChangedFilePicker lists workspace files the version-control system
// reports as changed
func NewChangedFilePicker(e *view.Editor) *Picker {
	return NewPicker(e, &changedFilePickerSource{
		Ident:       "changed-files",
		Label:       "Changed Files",
		Cols:        []string{"", ""},
		MatchCol:    1,
		Proportions: []int{0, 1},
	})
}

// Load lists the files changed against version control
func (c *changedFilePickerSource) Load(e *view.Editor) PickerLoad {
	vc := e.VersionControl()
	if vc == nil {
		return PickerLoad{Stop: func() {}}
	}
	changes, err := vc.ChangedFiles()
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return PickerLoad{Stop: func() {}}
	}
	// providers report symlink-resolved paths, so resolve the workspace root
	// the same way so names relativize cleanly
	cwd := e.Cwd()
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}

	nerd := e.Options().NerdFonts
	feed := make(chan *PickerItem)
	done := make(chan struct{})
	go func() {
		defer close(feed)
		for _, item := range changedFileRows(vc, changes, cwd, nerd) {
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
	return PickerLoad{Feed: feed, Stop: stop}
}

// Items returns the whole row set at once, letting a refresh swap the list
// in place rather than emptying and re-streaming it
func (c *changedFilePickerSource) Items(e *view.Editor) []*PickerItem {
	vc := e.VersionControl()
	if vc == nil {
		return nil
	}
	changes, err := vc.ChangedFiles()
	if err != nil {
		return nil
	}
	cwd := e.Cwd()
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}
	return changedFileRows(vc, changes, cwd, e.Options().NerdFonts)
}

// ItemsForPath returns the current VCS rows for path, one per stage, empty
// when it is no longer a changed file
func (c *changedFilePickerSource) ItemsForPath(
	e *view.Editor, path string,
) []*PickerItem {
	vc := e.VersionControl()
	if vc == nil {
		return nil
	}
	changes, err := vc.ChangedFiles()
	if err != nil {
		return nil
	}
	cwd := e.Cwd()
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}
	key := loader.CanonicalPath(path)
	nerd := e.Options().NerdFonts
	var out []*PickerItem
	for _, fc := range changes {
		if loader.CanonicalPath(fc.Path) == key {
			out = append(out, changedFileItem(changedFileItemArgs{
				vcs: vc, change: fc, cwd: cwd, nerd: nerd,
			}))
		}
	}
	return out
}

// Accept opens the chosen file
func (c *changedFilePickerSource) Accept(
	e *view.Editor, item *PickerItem, action PickerAcceptAction,
) {
	GotoPath(
		e, item.Location.Target.Path, GotoLines(item.Location.Lines), action,
	)
}

// HandleKey stages the selected file with ctrl+a and reverts it with ctrl+r,
// which unstages a staged row and discards the changes of an unstaged one
func (c *changedFilePickerSource) HandleKey(
	e *view.Editor, item *PickerItem, k command.KeyEvent,
) (BufferOverlayComponent, bool) {
	vc := e.VersionControl()
	if item == nil || vc == nil || k.Mods != command.ModCtrl {
		return nil, false
	}
	switch {
	case k.Code.Char == 'a':
		applyToRow(e, item, vc.Stage)
	case k.Code.Char != 'r':
		return nil, false
	case item.Location.Target.Variant == changedFileStaged:
		applyToRow(e, item, vc.Unstage)
	default:
		// discarding a working-tree change cannot be undone
		return confirmDiscard(item), true
	}
	return nil, true
}

func confirmDiscard(item *PickerItem) BufferOverlayComponent {
	question := i18n.Text(statusPickerDiscardKey, i18n.Vars{
		"file": item.Display,
	})
	return newConfirmation(question, func(e *view.Editor) tea.Cmd {
		if vc := e.VersionControl(); vc != nil {
			applyToRow(e, item, vc.Discard)
		}
		return nil
	}, true)
}

// a rename is one row over two paths, so both halves move together
func applyToRow(
	e *view.Editor, item *PickerItem, apply func(string) error,
) {
	paths := []string{item.Location.Target.Path}
	if item.BasePath != "" && item.BasePath != paths[0] {
		paths = append(paths, item.BasePath)
	}
	for _, path := range paths {
		if err := apply(path); err != nil {
			e.SetStatusMsg(i18n.ErrorText(err))
			return
		}
	}
}

func changedFileRows(
	vc view.VersionControl, changes []view.FileChange, cwd string, nerd bool,
) []*PickerItem {
	out := changedFileSections()
	var slab PickerItemSlab
	for _, fc := range changes {
		out = append(out, changedFileItem(changedFileItemArgs{
			slab:   &slab,
			vcs:    vc,
			change: fc,
			cwd:    cwd,
			nerd:   nerd,
		}))
	}
	return out
}

func changedFileSections() []*PickerItem {
	return []*PickerItem{
		{
			Display: i18n.Text(statusPickerStagedKey),
			Group:   changedFileStaged,
			Section: true,
		},
		{
			Display: i18n.Text(statusPickerUnstagedKey),
			Group:   changedFileUnstaged,
			Section: true,
		},
	}
}

type changedFileItemArgs struct {
	slab   *PickerItemSlab
	vcs    view.VersionControl
	change view.FileChange
	cwd    string
	nerd   bool
}

func changedFileItem(args changedFileItemArgs) *PickerItem {
	fc := args.change
	display := view.DocumentRelativeName(view.DocumentRelativeNameArgs{
		Path:    fc.Path,
		BaseDir: args.cwd,
	})
	// a rename shows only its destination, the source is in the diff preview
	lbl, sec := PickerNamePath(display)
	hunks := changedFileHunks(args.vcs, fc)
	basePath := fc.Path
	if fc.Kind == view.FileChangeRenamed {
		basePath = fc.FromPath
	}
	group := changedFileUnstaged
	if fc.Staged {
		group = changedFileStaged
	}
	item := PickerItem{
		Display:     display,
		Group:       group,
		Columns:     []string{changedFileIcon(fc.Kind, args.nerd), lbl},
		StyleScopes: []string{changedFileScope(fc.Kind), ""},
		SortKey:     display,
		SecFrom:     sec,
		DiffHunks:   hunks,
		DiffPreview: fc.Kind != view.FileChangeConflict,
		DiffKind:    fc.Kind,
		BasePath:    basePath,
		Location: PickerLocation{
			Target: PickerTarget{Path: fc.Path, Variant: group},
			Lines:  firstChangeLines(hunks),
		},
	}
	if args.slab != nil {
		return args.slab.Add(item)
	}
	return &item
}

func changedFileHunks(
	vc view.VersionControl, fc view.FileChange,
) []view.DiffHunk {
	switch fc.Kind {
	case view.FileChangeUntracked, view.FileChangeAdded,
		view.FileChangeDeleted:
		// the whole file is new or gone, so there is no base to diff against
		return nil
	default:
		if fc.Staged {
			return vc.StagedDiffHunks(fc.Path)
		}
		return vc.UnstagedDiffHunks(fc.Path)
	}
}

func firstChangeLines(hunks []view.DiffHunk) *core.Span {
	if len(hunks) == 0 {
		return nil
	}
	h := hunks[0]
	return &core.Span{From: h.From, To: max(h.From, h.To-1)}
}

func changedFileIcon(kind view.FileChangeKind, nerd bool) string {
	icons := changedFileIcons
	fallback := fileModifiedIcon
	if !nerd {
		icons = changedFileAscii
		fallback = "M"
	}
	if kind < 0 || int(kind) >= len(icons) {
		return fallback
	}
	return icons[kind]
}

func changedFileScope(kind view.FileChangeKind) string {
	if kind < 0 || int(kind) >= len(changedFileScopes) {
		return ""
	}
	return changedFileScopes[kind]
}

func lineRangeSelection(text core.Rope, lr *core.Span) (core.Selection, bool) {
	if lr == nil {
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
		[]core.Range{{Anchor: lineStart, Head: lineEnd}}, 0,
	)
	if err != nil {
		return core.Selection{}, false
	}
	return sel, true
}
