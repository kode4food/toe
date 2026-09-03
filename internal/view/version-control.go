package view

type (
	// VersionControl exposes version-control state to commands, pickers, and
	// rendering. Implementations live outside the view package. The editor only
	// holds the seam
	VersionControl interface {
		// DiffHunks returns the current hunks between the document and its
		// version-control base, sorted ascending and non-overlapping
		DiffHunks(*Document) []DiffHunk

		// DiffBase returns the version-control base text of the document
		DiffBase(*Document) (string, bool)

		// StagedDiffHunks computes hunks between the head and the staged
		// contents of an arbitrary workspace file
		StagedDiffHunks(path string) []DiffHunk

		// UnstagedDiffHunks computes hunks between the staged and the on-disk
		// contents of an arbitrary workspace file
		UnstagedDiffHunks(path string) []DiffHunk

		// HeadText returns the head text of an arbitrary workspace file, empty
		// when it has none
		HeadText(path string) string

		// IndexText returns the staged text of an arbitrary workspace file,
		// falling back to its head text when it has no entry in the index
		IndexText(path string) string

		// HeadName returns a short display name for the current head of the
		// repository containing the document
		HeadName(*Document) (string, bool)

		// ChangedFiles lists workspace files that differ from the head
		ChangedFiles() ([]FileChange, error)

		// Stage adds the working-tree state of path to the staging area
		Stage(path string) error

		// Unstage drops path from the staging area, leaving the working tree
		Unstage(path string) error

		// Discard restores the working-tree state of path from the staging
		// area, deleting the file when version control does not track it
		Discard(path string) error

		// Ignore adds path to the ignore list of the repository holding it
		Ignore(path string) error

		// Refresh picks up external version-control state changes
		Refresh()

		// Updates delivers a token whenever diff state changes, so the UI can
		// schedule a redraw
		Updates() <-chan struct{}
	}

	// DiffHunk is a change as half-open ranges [BaseFrom,BaseTo) and [From,To).
	// An empty base range is a pure insertion, empty doc range a pure removal
	DiffHunk struct {
		BaseFrom int
		BaseTo   int
		From     int
		To       int
	}

	// FileChange describes one change to one file reported by version control.
	// A file edited both in the index and the working tree yields two changes,
	// one per stage
	FileChange struct {
		Kind     FileChangeKind
		Path     string
		FromPath string // original path, set only for FileChangeRenamed
		Staged   bool
	}

	// FileChangeKind classifies a FileChange
	FileChangeKind int
)

const (
	FileChangeUntracked FileChangeKind = iota
	FileChangeAdded
	FileChangeModified
	FileChangeConflict
	FileChangeDeleted
	FileChangeRenamed
)

// PureInsertion reports whether the hunk only adds document lines
func (h DiffHunk) PureInsertion() bool {
	return h.BaseFrom == h.BaseTo
}

// PureRemoval reports whether the hunk only removes base lines
func (h DiffHunk) PureRemoval() bool {
	return h.From == h.To
}

// SetVersionControl installs the version-control state provider
func (e *Editor) SetVersionControl(vc VersionControl) {
	e.versionControl = vc
}

// VersionControl returns the installed version-control state provider
func (e *Editor) VersionControl() VersionControl {
	return e.versionControl
}
