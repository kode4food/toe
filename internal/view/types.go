package view

import (
	"fmt"
	"path/filepath"
	"strings"
)

type (
	// DocumentId is the unique identifier for an open document
	DocumentId int

	// Id is the unique identifier for an open view
	Id int

	// Mode describes the current editing mode
	Mode int

	// Layout describes how child views are arranged within a split container
	Layout int

	// Direction is used to navigate between splits
	Direction int

	// Align describes vertical scroll alignment
	Align int

	// Area is the screen rectangle assigned to a view by the layout engine
	Area struct {
		X, Y, Width, Height int
	}

	// Position holds the scroll offset for a view
	Position struct {
		// Anchor is the first visible char position in the document
		Anchor int
		// HorizontalOffset is the number of columns scrolled right
		HorizontalOffset int
		// VerticalOffset is lines of context above the visible area
		VerticalOffset int
	}

	// JumpList manages a bounded history of cursor positions
	JumpList struct {
		items []jump
		head  int
	}

	jump struct {
		docID  DocumentId
		anchor int
	}

	// JumpEntry is a single entry in the jump history
	JumpEntry struct {
		DocID  DocumentId
		Anchor int
	}

	// DocumentSavedEvent carries information about a successfully saved doc
	DocumentSavedEvent struct {
		DocID DocumentId
		Path  string
	}

	// DocumentOpenError describes why a document could not be opened
	DocumentOpenError struct {
		Path string
		Err  error
	}
)

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeSelect
)

const (
	// LayoutVertical places splits side by side
	LayoutVertical Layout = iota
	// LayoutHorizontal stacks splits one above the other
	LayoutHorizontal
)

const (
	DirectionUp Direction = iota
	DirectionDown
	DirectionLeft
	DirectionRight
)

const (
	// InvalidDocumentId is the zero value, indicating no document
	InvalidDocumentId DocumentId = 0
	// InvalidViewId is the zero value, indicating no view
	InvalidViewId Id = 0
	// ScratchBufferName is the display name used for unnamed scratch documents
	ScratchBufferName = "[scratch]"

	jumpListCap = 64
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NOR"
	case ModeInsert:
		return "INS"
	case ModeSelect:
		return "SEL"
	}
	return "NOR"
}

// DocumentDisplayName returns a short display name for a file path,
// or ScratchBufferName if path is empty
func DocumentDisplayName(path string) string {
	if path == "" {
		return ScratchBufferName
	}
	return filepath.Base(path)
}

// DocumentRelativeName returns path relative to basedir,
// falling back to the absolute path on error
func DocumentRelativeName(path, basedir string) string {
	if path == "" {
		return ScratchBufferName
	}
	rel, err := filepath.Rel(basedir, path)
	if err != nil {
		return path
	}
	if !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

// Entries returns all jump history entries from oldest to newest
func (j *JumpList) Entries() []JumpEntry {
	out := make([]JumpEntry, len(j.items))
	for i, it := range j.items {
		out[i] = JumpEntry{it.docID, it.anchor}
	}
	return out
}

// Push adds a new jump position, discarding forward history
func (j *JumpList) Push(docID DocumentId, anchor int) {
	if len(j.items) > 0 && j.head < len(j.items) {
		j.items = j.items[:j.head]
	}
	j.items = append(j.items, jump{docID, anchor})
	if len(j.items) > jumpListCap {
		j.items = j.items[len(j.items)-jumpListCap:]
	}
	j.head = len(j.items)
}

// Backward moves to the previous jump and returns it
func (j *JumpList) Backward() (DocumentId, int, bool) {
	if j.head <= 1 {
		return 0, 0, false
	}
	j.head--
	it := j.items[j.head-1]
	return it.docID, it.anchor, true
}

// Forward moves to the next jump and returns it
func (j *JumpList) Forward() (DocumentId, int, bool) {
	if j.head >= len(j.items) {
		return 0, 0, false
	}
	it := j.items[j.head]
	j.head++
	return it.docID, it.anchor, true
}

func (d *DocumentOpenError) Error() string {
	return fmt.Sprintf("open %s: %v", d.Path, d.Err)
}

func (d *DocumentOpenError) Unwrap() error {
	return d.Err
}
