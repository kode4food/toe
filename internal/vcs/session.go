// Package vcs integrates version-control systems with the editor: diff bases
// for gutter hunks, changed-file listings, and head names. Git is the only
// provider today. The Provider interface keeps the API explicit and pluggable
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
		headIDs  map[view.DocumentId]string
		loading  map[view.DocumentId]bool
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

		// HeadID returns the exact current head revision
		HeadID(path string) (string, error)

		// ChangedFiles lists workspace files that differ from the head
		ChangedFiles(cwd string) ([]view.FileChange, error)

		// IndexText returns the staged contents of path
		IndexText(cwd, path string) ([]byte, error)

		// Stage adds the working-tree state of path to the staging area of the
		// repository containing cwd
		Stage(cwd, path string) error

		// Unstage drops path from that staging area, leaving the working tree
		Unstage(cwd, path string) error
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
		headIDs:  map[view.DocumentId]string{},
		loading:  map[view.DocumentId]bool{},
		updates:  make(chan struct{}, 1),
	}
	e.AddDocumentObserver(s)
	e.SetVersionControl(s)
	for _, doc := range e.VisibleDocuments() {
		s.DocumentOpened(doc)
	}
	return s
}

// DiffHunks returns the current hunks for the document, loading its diff base
// on first request for a buffer that was not open at startup
func (s *Session) DiffHunks(doc *view.Document) []view.DiffHunk {
	if d := s.differ(doc); d != nil {
		return d.Hunks()
	}
	s.ensureDiffBase(doc)
	return nil
}

// DiffBase returns the version-control base text for the document
func (s *Session) DiffBase(doc *view.Document) (string, bool) {
	if d := s.differ(doc); d != nil {
		return d.Base().String(), true
	}
	return "", false
}

// StagedDiffHunks computes hunks between the head and the staged contents of an
// arbitrary workspace file. It shells out to the provider, so it is intended
// for on-demand use such as picker previews
func (s *Session) StagedDiffHunks(path string) []view.DiffHunk {
	return Diff(DiffSides{
		Base: core.NewRope(s.HeadText(path)),
		Doc:  core.NewRope(s.IndexText(path)),
	})
}

// UnstagedDiffHunks computes hunks between the staged and the on-disk contents
// of an arbitrary workspace file. It shells out to the provider
func (s *Session) UnstagedDiffHunks(path string) []view.DiffHunk {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return Diff(DiffSides{
		Base: core.NewRope(s.IndexText(path)),
		Doc:  baseRope(data),
	})
}

// HeadText returns the head text of an arbitrary workspace file. It shells out
// to the provider, so it is intended for on-demand use such as picker previews
func (s *Session) HeadText(path string) string {
	if base, err := s.provider.DiffBase(path); err == nil {
		return baseRope(base).String()
	}
	return ""
}

// IndexText returns the staged text of an arbitrary workspace file, falling
// back to its head text when the file has no entry in the index
func (s *Session) IndexText(path string) string {
	data, err := s.provider.IndexText(s.editor.Cwd(), path)
	if err != nil {
		return s.HeadText(path)
	}
	return baseRope(data).String()
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

// Stage adds the working-tree state of path to the staging area
func (s *Session) Stage(path string) error {
	return s.provider.Stage(s.editor.Cwd(), path)
}

// Unstage drops path from the staging area, leaving the working tree
func (s *Session) Unstage(path string) error {
	return s.provider.Unstage(s.editor.Cwd(), path)
}

// Refresh reloads open document diff bases when their repository head moves
func (s *Session) Refresh() {
	for _, doc := range s.editor.AllDocuments() {
		path := doc.Path()
		if path == "" || !s.headMoved(doc, path) {
			continue
		}
		if s.differ(doc) == nil {
			continue
		}
		text := doc.Text()
		go s.loadDiffBase(doc, path, text)
	}
}

// Updates delivers a token whenever diff state changes
func (s *Session) Updates() <-chan struct{} {
	return s.updates
}

// DocumentOpened fetches the diff base in the background and starts a differ
// once it arrives
func (s *Session) DocumentOpened(doc *view.Document) {
	s.ensureDiffBase(doc)
}

// DocumentChanged feeds the new document text to the differ
func (s *Session) DocumentChanged(doc *view.Document, _ view.DocumentChange) {
	if d := s.differ(doc); d != nil {
		d.SetDoc(doc.Text())
	}
}

// DocumentSaved refreshes the diff base. The head may have moved since the
// document was opened
func (s *Session) DocumentSaved(doc *view.Document) {
	path := doc.Path()
	if path == "" {
		return
	}
	text := doc.Text()
	go s.loadDiffBase(doc, path, text)
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
	delete(s.headIDs, doc.ID())
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

func (s *Session) ensureDiffBase(doc *view.Document) {
	path := doc.Path()
	if path == "" {
		return
	}
	id := doc.ID()
	s.mu.Lock()
	_, hasDiffer := s.differs[id]
	if hasDiffer || s.loading[id] {
		s.mu.Unlock()
		return
	}
	s.loading[id] = true
	s.mu.Unlock()
	text := doc.Text()
	go func() {
		s.loadDiffBase(doc, path, text)
		s.mu.Lock()
		delete(s.loading, id)
		s.mu.Unlock()
	}()
}

func (s *Session) differ(doc *view.Document) *Differ {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.differs[doc.ID()]
}

func (s *Session) headMoved(doc *view.Document, path string) bool {
	head, err := s.provider.HeadID(path)
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.headIDs[doc.ID()]
	return ok && old != head
}

func (s *Session) loadDiffBase(
	doc *view.Document, path string, text core.Rope,
) {
	base, err := s.provider.DiffBase(path)
	if err != nil {
		return
	}
	rope := baseRope(base)
	name, _ := s.provider.HeadName(path)
	head, _ := s.provider.HeadID(path)

	s.mu.Lock()
	if d, ok := s.differs[doc.ID()]; ok {
		s.heads[doc.ID()] = name
		s.headIDs[doc.ID()] = head
		s.mu.Unlock()
		d.SetBase(rope)
		return
	}
	d := NewDiffer(DiffSides{Base: rope, Doc: text}, s.notifyUpdate)
	s.differs[doc.ID()] = d
	s.heads[doc.ID()] = name
	s.headIDs[doc.ID()] = head
	s.mu.Unlock()
	s.notifyUpdate()
}

func (s *Session) notifyUpdate() {
	select {
	case s.updates <- struct{}{}:
	default:
	}
}

func baseRope(data []byte) core.Rope {
	return core.NewRope(strings.TrimPrefix(string(data), "\ufeff"))
}
