package view

import (
	"os"
	"path/filepath"
)

// Save saves the focused document to disk
func (e *Editor) Save() error {
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
	if err := doc.Save(&e.opts); err != nil {
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
	e.docs[doc.ID()] = doc
	e.documentOpened(doc)
	v, ok := e.FocusedView()
	if ok {
		v.docID = doc.ID()
		v.offset = Position{}
		e.markDocAccessed()
		return v
	}
	nv := &View{docID: doc.ID(), mode: ModeNormal}
	e.tree.Insert(nv)
	e.markDocAccessed()
	return nv
}

// SaveAll saves all modified documents
func (e *Editor) SaveAll() []error {
	var errs []error
	for _, doc := range e.docs {
		if doc.Modified() {
			creating := fileMissing(doc.Path())
			if ops, ok := e.fileOperationController(); ok && creating {
				_ = ops.WillCreateFile(doc.Path(), false)
			}
			before := doc.Text()
			rev := doc.Revision()
			if err := doc.Save(&e.opts); err != nil {
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
		return e.Save()
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
		return e.Save()
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
	for _, doc := range e.docs {
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
	e.cwd = abs
	return nil
}

// PushDirectory pushes the current directory onto the stack then chdirs
func (e *Editor) PushDirectory(path string) error {
	e.dirStack = append(e.dirStack, e.cwd)
	return e.Chdir(path)
}

// PopDirectory changes to the top of the directory stack, if any
func (e *Editor) PopDirectory() error {
	if len(e.dirStack) == 0 {
		return ErrEmptyDirStack
	}
	top := e.dirStack[len(e.dirStack)-1]
	e.dirStack = e.dirStack[:len(e.dirStack)-1]
	return e.Chdir(top)
}

// DirStack returns a copy of the directory stack (bottom to top)
func (e *Editor) DirStack() []string {
	cp := make([]string, len(e.dirStack))
	copy(cp, e.dirStack)
	return cp
}

// OpenFile replaces the focused view's document with the given file
func (e *Editor) OpenFile(path string) (*View, error) {
	return e.SwitchFile(path)
}

// SwitchFile replaces the focused view's document with the given file, reusing
// an existing document if it is already open
func (e *Editor) SwitchFile(path string) (*View, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	for _, d := range e.docs {
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
	e.docs[doc.ID()] = doc
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
	if _, ok := e.docs[did]; !ok {
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

// SwitchOrOpenDoc returns an existing document for path, opening it if needed
func (e *Editor) SwitchOrOpenDoc(path string) (*Document, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	for _, d := range e.docs {
		if d.Path() == absPath {
			return d, nil
		}
	}
	doc, err := e.openFile(absPath)
	if err != nil {
		return nil, err
	}
	e.docs[doc.ID()] = doc
	e.documentOpened(doc)
	return doc, nil
}

func (e *Editor) newDocument() *Document {
	e.nextDocID++
	return newDocument(e.nextDocID, &e.opts)
}

func (e *Editor) openFile(path string) (*Document, error) {
	e.nextDocID++
	return openDocument(e.nextDocID, path, &e.opts)
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
