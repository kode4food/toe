package view

import (
	"os"
	"path/filepath"
)

// Save saves the focused document to disk. Unless force is set, it refuses
// an unsafe overwrite (changed on disk, or read-only)
func (e *Editor) Save(force bool) error {
	doc, ok := e.FocusedDocument()
	if !ok {
		return ErrNoDocument
	}
	creating := fileMissing(doc.Path())
	if ops, ok := e.fileOperationController(); ok && creating {
		_ = ops.WillCreateFile(doc.Path(), false)
	}
	before := doc.Text()
	rev := doc.Revision()
	if err := doc.Save(&e.opts, force); err != nil {
		return err
	}
	if doc.Revision() != rev {
		e.documentChanged(doc, wholeDocumentChange(before, doc.Text().String()))
	}
	e.documentSaved(doc)
	if ops, ok := e.fileOperationController(); ok && creating {
		_ = ops.DidCreateFile(doc.Path(), false)
	}
	return nil
}

// NewDocument creates a new empty scratch document and makes it the focused
// view
func (e *Editor) NewDocument() *View {
	doc := e.newDocument()
	e.documents.byID[doc.ID()] = doc
	e.documentOpened(doc)
	v, ok := e.FocusedView()
	if ok {
		v.docID = doc.ID()
		v.offset = Position{}
		e.markDocAccessed()
		return v
	}
	nv := &View{editor: e, docID: doc.ID(), mode: ModeNormal}
	if e.panes.tree.Get(e.panes.tree.Focus()) == nil {
		e.panes.tree.Insert(nv)
		e.markDocAccessed()
		return nv
	}
	old := e.ReplacePane(e.panes.tree.Focus(), nv)
	e.DiscardPane(old)
	e.markDocAccessed()
	return nv
}

// SaveAll saves all modified documents. Unless force is set, it refuses an
// unsafe overwrite (changed on disk, or read-only)
func (e *Editor) SaveAll(force bool) []error {
	var errs []error
	for _, doc := range e.documents.byID {
		if doc.Modified() {
			creating := fileMissing(doc.Path())
			if ops, ok := e.fileOperationController(); ok && creating {
				_ = ops.WillCreateFile(doc.Path(), false)
			}
			before := doc.Text()
			rev := doc.Revision()
			if err := doc.Save(&e.opts, force); err != nil {
				errs = append(errs, err)
				continue
			}
			if doc.Revision() != rev {
				change := wholeDocumentChange(before, doc.Text().String())
				e.documentChanged(doc, change)
			}
			e.documentSaved(doc)
			if ops, ok := e.fileOperationController(); ok && creating {
				_ = ops.DidCreateFile(doc.Path(), false)
			}
		}
	}
	return errs
}

// MoveFocusedFile renames the focused document's backing file and updates the
// document path
func (e *Editor) MoveFocusedFile(path string, force bool) error {
	doc, ok := e.FocusedDocument()
	if !ok {
		return ErrNoDocument
	}
	if doc.Modified() && !force {
		return ErrUnsavedChanges
	}
	oldPath := doc.Path()
	if oldPath == "" || fileMissing(oldPath) {
		doc.SetPath(path)
		return e.Save(force)
	}
	newPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	oldAbs, err := filepath.Abs(oldPath)
	if err != nil {
		return err
	}
	if oldAbs == newPath {
		return nil
	}
	if ops, ok := e.fileOperationController(); ok {
		_ = ops.WillRenameFile(oldAbs, newPath, false)
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(oldAbs, newPath); err != nil {
		return err
	}
	doc.SetPath(newPath)
	if ops, ok := e.fileOperationController(); ok {
		_ = ops.DidRenameFile(oldAbs, newPath, false)
	}
	if doc.Modified() {
		return e.Save(force)
	}
	return nil
}

// Reload reloads the focused document from disk
func (e *Editor) Reload() error {
	doc, ok := e.FocusedDocument()
	if !ok {
		return ErrNoDocument
	}
	return doc.Reload()
}

// ReloadAll reloads all documents that have a file path
func (e *Editor) ReloadAll() []error {
	var errs []error
	for _, doc := range e.documents.byID {
		if doc.Path() != "" {
			if err := doc.Reload(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// Chdir changes the editor working directory
func (e *Editor) Chdir(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.Chdir(abs); err != nil {
		return err
	}
	e.workspace.cwd = abs
	return nil
}

// PushDirectory pushes the current directory onto the stack then chdirs
func (e *Editor) PushDirectory(path string) error {
	e.workspace.dirStack = append(e.workspace.dirStack, e.workspace.cwd)
	return e.Chdir(path)
}

// PopDirectory changes to the top of the directory stack, if any
func (e *Editor) PopDirectory() error {
	if len(e.workspace.dirStack) == 0 {
		return ErrEmptyDirStack
	}
	top := e.workspace.dirStack[len(e.workspace.dirStack)-1]
	e.workspace.dirStack = e.workspace.dirStack[:len(e.workspace.dirStack)-1]
	return e.Chdir(top)
}

// DirStack returns a copy of the directory stack (bottom to top)
func (e *Editor) DirStack() []string {
	cp := make([]string, len(e.workspace.dirStack))
	copy(cp, e.workspace.dirStack)
	return cp
}

// OpenFile replaces the focused view's document with the given file, reusing
// an existing document if it is already open
func (e *Editor) OpenFile(path string) (*View, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	for _, d := range e.documents.byID {
		if d.Path() == absPath {
			e.recordPrevDoc()
			if v, ok := e.FocusedView(); ok {
				v.docID = d.ID()
				v.offset = Position{}
				e.markDocAccessed()
				return v, nil
			}
			return nil, ErrNoView
		}
	}

	doc, err := e.openFile(absPath)
	if err != nil {
		return nil, err
	}
	e.recordPrevDoc()
	e.documents.byID[doc.ID()] = doc
	e.documentOpened(doc)
	if v, ok := e.FocusedView(); ok {
		v.docID = doc.ID()
		v.offset = Position{}
		e.markDocAccessed()
		return v, nil
	}
	return nil, ErrNoView
}

// SwitchBuffer replaces the focused view's document with an already-open
// document by ID. Returns false if the document does not exist or there is no
// focused view
func (e *Editor) SwitchBuffer(did DocumentId) bool {
	if _, ok := e.documents.byID[did]; !ok {
		return false
	}
	e.recordPrevDoc()
	if v, ok := e.FocusedView(); ok {
		v.docID = did
		v.offset = Position{}
		e.markDocAccessed()
		return true
	}
	return false
}

// ShowDocument displays an open document in the focused pane
func (e *Editor) ShowDocument(did DocumentId) (*View, bool) {
	if e.SwitchBuffer(did) {
		return e.FocusedView()
	}
	if _, ok := e.documents.byID[did]; !ok {
		return nil, false
	}
	id := e.panes.tree.Focus()
	if e.panes.tree.Get(id) == nil {
		return nil, false
	}
	v := &View{editor: e, docID: did, mode: ModeNormal}
	old := e.ReplacePane(id, v)
	e.DiscardPane(old)
	e.markDocAccessed()
	return v, true
}

// SwitchOrOpenDoc returns an existing document for path, opening it if needed
func (e *Editor) SwitchOrOpenDoc(path string) (*Document, error) {
	doc, err := e.PeekDoc(path)
	if err != nil {
		return nil, err
	}
	if _, ok := e.documents.byID[doc.ID()]; !ok {
		e.documents.byID[doc.ID()] = doc
		e.documentOpened(doc)
	}
	return doc, nil
}

// PeekDoc reads path without registering it as a buffer
func (e *Editor) PeekDoc(path string) (*Document, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	for _, d := range e.documents.byID {
		if d.Path() == absPath {
			return d, nil
		}
	}
	return e.openFile(absPath)
}

func (e *Editor) newDocument() *Document {
	e.documents.nextID++
	return newDocument(e.documents.nextID, &e.opts)
}

func (e *Editor) openFile(path string) (*Document, error) {
	e.documents.nextID++
	return openDocument(e.documents.nextID, path, &e.opts)
}

func (e *Editor) fileOperationController() (FileOperationController, bool) {
	ops, ok := e.langServers.(FileOperationController)
	return ops, ok
}

func fileMissing(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}
