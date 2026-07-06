// Package vcs integrates version-control systems with the editor: diff bases
// for gutter hunks, changed-file listings, and head names. Git is the only
// provider today; the Provider interface keeps the API explicit and pluggable
package vcs

import (
	"os"
	"strings"
	"sync"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Session owns a differ per open document and implements the editor's
	// VersionControl seam over the version-control provider
	Session struct {
		mu       sync.Mutex
		editor   *view.Editor
		provider Provider
		differs  map[view.DocumentId]*Differ
		heads    map[view.DocumentId]string
		updates  chan struct{}
	}

	// Provider supplies diff bases and change information from one
	// version-control system
	Provider interface {
		// DiffBase returns the checked-in contents of path, the "base" text a
		// diff of the working copy is computed against
		DiffBase(path string) ([]byte, error)

		// HeadName returns a short display name for the current head, such as
		// a branch name
		HeadName(path string) (string, error)

		// ChangedFiles lists workspace files that differ from the head
		ChangedFiles(cwd string) ([]view.FileChange, error)
	}
)

var (
	_ view.DocumentObserver = (*Session)(nil)
	_ view.VersionControl   = (*Session)(nil)
)

// Attach starts a version-control session for the editor, observing document
// lifecycle events and serving diff state
func Attach(e *view.Editor) *Session {
	s := &Session{
		editor:   e,
		provider: Git{},
		differs:  map[view.DocumentId]*Differ{},
		heads:    map[view.DocumentId]string{},
		updates:  make(chan struct{}, 1),
	}
	e.AddDocumentObserver(s)
	e.SetVersionControl(s)
	for _, doc := range e.AllDocuments() {
		s.DocumentOpened(doc)
	}
	return s
}

// DiffHunks returns the current hunks for the document
func (s *Session) DiffHunks(doc *view.Document) []view.DiffHunk {
	if d, ok := s.differ(doc); ok {
		return d.Hunks()
	}
	return nil
}

// DiffBase returns the version-control base text for the document
func (s *Session) DiffBase(doc *view.Document) (string, bool) {
	if d, ok := s.differ(doc); ok {
		return d.Base().String(), true
	}
	return "", false
}

// DiffHunksForPath computes hunks between the checked-in base and the on-disk
// contents of an arbitrary workspace file. It shells out to the provider and is
// intended for on-demand use such as picker previews
func (s *Session) DiffHunksForPath(path string) []view.DiffHunk {
	base, err := s.provider.DiffBase(path)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return Diff(baseRope(base), baseRope(data))
}

// HeadName returns the head display name for the document's repository
func (s *Session) HeadName(doc *view.Document) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, ok := s.heads[doc.ID()]
	return name, ok
}

// ChangedFiles lists workspace files that differ from the head
func (s *Session) ChangedFiles() ([]view.FileChange, error) {
	return s.provider.ChangedFiles(s.editor.Cwd())
}

// Updates delivers a token whenever diff state changes
func (s *Session) Updates() <-chan struct{} {
	return s.updates
}

// DocumentOpened fetches the diff base in the background and starts a differ
// once it arrives
func (s *Session) DocumentOpened(doc *view.Document) {
	path := doc.Path()
	if path == "" {
		return
	}
	go s.loadDiffBase(doc, path)
}

// DocumentChanged feeds the new document text to the differ
func (s *Session) DocumentChanged(doc *view.Document, _ view.DocumentChange) {
	if d, ok := s.differ(doc); ok {
		d.SetDoc(doc.Text())
	}
}

// DocumentSaved refreshes the diff base; the head may have moved since the
// document was opened
func (s *Session) DocumentSaved(doc *view.Document) {
	path := doc.Path()
	if path == "" {
		return
	}
	go s.loadDiffBase(doc, path)
}

// DocumentClosed stops and forgets the document's differ
func (s *Session) DocumentClosed(doc *view.Document) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.differs[doc.ID()]; ok {
		d.Close()
		delete(s.differs, doc.ID())
	}
	delete(s.heads, doc.ID())
}

// Close stops all differs
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, d := range s.differs {
		d.Close()
		delete(s.differs, id)
	}
}

func (s *Session) differ(doc *view.Document) (*Differ, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.differs[doc.ID()]
	return d, ok
}

// loadDiffBase resolves the diff base and head name for doc, creating or
// updating its differ. It runs off the main goroutine because providers may
// shell out
func (s *Session) loadDiffBase(doc *view.Document, path string) {
	base, err := s.provider.DiffBase(path)
	if err != nil {
		return
	}
	rope := baseRope(base)
	name, _ := s.provider.HeadName(path)

	s.mu.Lock()
	if d, ok := s.differs[doc.ID()]; ok {
		s.heads[doc.ID()] = name
		s.mu.Unlock()
		d.SetBase(rope)
		return
	}
	d := NewDiffer(rope, doc.Text(), s.notifyUpdate)
	s.differs[doc.ID()] = d
	s.heads[doc.ID()] = name
	s.mu.Unlock()
	s.notifyUpdate()
}

// notifyUpdate coalesces update tokens; a full channel already implies a
// pending redraw
func (s *Session) notifyUpdate() {
	select {
	case s.updates <- struct{}{}:
	default:
	}
}

func baseRope(data []byte) core.Rope {
	return core.NewRope(strings.TrimPrefix(string(data), "\ufeff"))
}
