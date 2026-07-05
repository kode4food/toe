package view

type (
	// VersionControl exposes version-control state to commands, pickers, and
	// rendering. Implementations live outside the view package; the editor only
	// holds the seam
	VersionControl interface {
		// DiffHunks returns the current hunks between the document and its
		// version-control base, sorted ascending and non-overlapping
		DiffHunks(*Document) []DiffHunk

		// DiffBase returns the version-control base text of the document
		DiffBase(*Document) (string, bool)

		// DiffHunksForPath computes hunks between the checked-in base and the
		// on-disk contents of an arbitrary workspace file
		DiffHunksForPath(path string) []DiffHunk

		// HeadName returns a short display name for the current head of the
		// repository containing the document
		HeadName(*Document) (string, bool)

		// ChangedFiles lists workspace files that differ from the head
		ChangedFiles() ([]FileChange, error)

		// Updates delivers a token whenever diff state changes, so the UI can
		// schedule a redraw
		Updates() <-chan struct{}
	}

	// DiffHunk is one contiguous change between the version-control base and
	// the document, as half-open line ranges [BaseFrom,BaseTo) and [From,To).
	// A pure insertion has an empty base range; a pure removal has an empty
	// document range
	DiffHunk struct {
		BaseFrom int
		BaseTo   int
		From     int
		To       int
	}

	// FileChange describes one changed file reported by version control
	FileChange struct {
		Kind     FileChangeKind
		Path     string
		FromPath string // original path, set only for FileChangeRenamed
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
