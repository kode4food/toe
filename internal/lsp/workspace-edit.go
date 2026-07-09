package lsp

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type workspaceDocumentEdit struct {
	doc     *view.Document
	changes []core.Change
}

var (
	ErrWorkspaceEditUnsupported = errors.New("workspace edit unsupported")
	ErrWorkspaceEditURI         = errors.New("workspace edit URI unsupported")
	ErrWorkspaceEditRange       = errors.New("workspace edit range invalid")
	ErrWorkspaceEditFile        = errors.New(
		"workspace edit file operation failed",
	)
)

// ApplyWorkspaceEdit applies a server-provided workspace edit
func (s *Session) ApplyWorkspaceEdit(
	server string, edit protocol.WorkspaceEdit,
) error {
	client, ok := s.servers.client(server)
	if !ok {
		return ErrUnknownLanguageServer
	}
	return s.applyWorkspaceEdit(edit, client.OffsetEncoding())
}

func (s *Session) applyWorkspaceEdit(
	edit protocol.WorkspaceEdit,
	encoding protocol.PositionEncodingKind,
) error {
	if len(edit.DocumentChanges) > 0 {
		return s.applyDocumentChanges(edit.DocumentChanges, encoding)
	}
	return s.applyWorkspaceChanges(edit.Changes, encoding)
}

func (s *Session) applyWorkspaceChanges(
	changes map[uri.URI][]protocol.TextEdit,
	encoding protocol.PositionEncodingKind,
) error {
	uris := make([]uri.URI, 0, len(changes))
	for u := range changes {
		uris = append(uris, u)
	}
	slices.SortFunc(uris, func(a, b uri.URI) int {
		return cmp.Compare(a.String(), b.String())
	})
	for _, u := range uris {
		docEdit, err := s.workspaceDocumentEdit(
			u, changes[u], encoding,
		)
		if err != nil {
			return err
		}
		if err := s.applyWorkspaceDocumentEdit(docEdit); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) applyDocumentChanges(
	changes []protocol.DocumentChange,
	encoding protocol.PositionEncodingKind,
) error {
	for _, change := range changes {
		switch c := change.(type) {
		case *protocol.TextDocumentEdit:
			docEdit, err := s.textDocumentEdit(c, encoding)
			if err != nil {
				return err
			}
			if err := s.applyWorkspaceDocumentEdit(docEdit); err != nil {
				return err
			}
		case *protocol.CreateFile:
			if err := s.applyCreateFile(c); err != nil {
				return err
			}
		case *protocol.RenameFile:
			if err := s.applyRenameFile(c); err != nil {
				return err
			}
		case *protocol.DeleteFile:
			if err := s.applyDeleteFile(c); err != nil {
				return err
			}
		default:
			return fmt.Errorf(
				"%w: document change", ErrWorkspaceEditUnsupported,
			)
		}
	}
	return nil
}

func (s *Session) applyCreateFile(op *protocol.CreateFile) error {
	path, err := workspaceEditPath(op.URI)
	if err != nil {
		return err
	}
	exists, err := pathExists(path)
	if err != nil {
		return err
	}
	overwrite := op.Options != nil && boolValue(op.Options.Overwrite)
	ignore := op.Options != nil && boolValue(op.Options.IgnoreIfExists)
	if exists && ignore {
		return nil
	}
	if exists && !overwrite {
		return fmt.Errorf("%w: %s", ErrWorkspaceEditFile, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%w: %v", ErrWorkspaceEditFile, err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		return fmt.Errorf("%w: %v", ErrWorkspaceEditFile, err)
	}
	s.didChangeWatchedFile(path)
	return nil
}

func (s *Session) applyRenameFile(op *protocol.RenameFile) error {
	oldPath, err := workspaceEditPath(op.OldURI)
	if err != nil {
		return err
	}
	newPath, err := workspaceEditPath(op.NewURI)
	if err != nil {
		return err
	}
	skip, err := prepareRenameTarget(newPath, op.Options)
	if err != nil || skip {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return fmt.Errorf("%w: %v", ErrWorkspaceEditFile, err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("%w: %v", ErrWorkspaceEditFile, err)
	}
	s.renameOpenDocument(oldPath, newPath)
	s.didChangeWatchedFile(oldPath)
	s.didChangeWatchedFile(newPath)
	return nil
}

func (s *Session) applyDeleteFile(op *protocol.DeleteFile) error {
	path, err := workspaceEditPath(op.URI)
	if err != nil {
		return err
	}
	exists, err := pathExists(path)
	if err != nil {
		return err
	}
	ignore := op.Options != nil && boolValue(op.Options.IgnoreIfNotExists)
	if !exists && ignore {
		return nil
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrWorkspaceEditFile, path)
	}
	recursive := op.Options != nil && boolValue(op.Options.Recursive)
	if recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWorkspaceEditFile, err)
	}
	s.didChangeWatchedFile(path)
	return nil
}

func (s *Session) workspaceDocumentEdit(
	u uri.URI, edits []protocol.TextEdit,
	encoding protocol.PositionEncodingKind,
) (workspaceDocumentEdit, error) {
	doc, err := s.documentForURI(u)
	if err != nil {
		return workspaceDocumentEdit{}, err
	}
	changes, err := textEditsToChanges(doc, edits, encoding)
	if err != nil {
		return workspaceDocumentEdit{}, err
	}
	return workspaceDocumentEdit{doc: doc, changes: changes}, nil
}

func (s *Session) textDocumentEdit(
	edit *protocol.TextDocumentEdit,
	encoding protocol.PositionEncodingKind,
) (workspaceDocumentEdit, error) {
	doc, err := s.documentForURI(edit.TextDocument.URI)
	if err != nil {
		return workspaceDocumentEdit{}, err
	}
	edits := make([]protocol.TextEdit, 0, len(edit.Edits))
	for _, elem := range edit.Edits {
		switch e := elem.(type) {
		case *protocol.TextEdit:
			edits = append(edits, *e)
		case *protocol.AnnotatedTextEdit:
			edits = append(edits, e.TextEdit)
		case *protocol.SnippetTextEdit:
			return workspaceDocumentEdit{}, fmt.Errorf(
				"%w: snippet text edit", ErrWorkspaceEditUnsupported,
			)
		default:
			return workspaceDocumentEdit{}, fmt.Errorf(
				"%w: text document edit", ErrWorkspaceEditUnsupported,
			)
		}
	}
	changes, err := textEditsToChanges(doc, edits, encoding)
	if err != nil {
		return workspaceDocumentEdit{}, err
	}
	return workspaceDocumentEdit{doc: doc, changes: changes}, nil
}

func (s *Session) applyWorkspaceDocumentEdit(edit workspaceDocumentEdit) error {
	if len(edit.changes) == 0 {
		return nil
	}
	cs, err := core.NewChangeSetFromChanges(edit.doc.Text(), edit.changes)
	if err != nil {
		return err
	}
	tx := core.NewTransaction(edit.doc.Text()).WithChanges(cs)
	return s.editor.ApplyToDocument(edit.doc, tx)
}

func (s *Session) documentForURI(u uri.URI) (*view.Document, error) {
	path, err := workspaceEditPath(u)
	if err != nil {
		return nil, err
	}
	doc, err := s.editor.SwitchOrOpenDoc(path)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func workspaceEditPath(u uri.URI) (string, error) {
	if !u.IsFile() {
		return "", fmt.Errorf("%w: %s", ErrWorkspaceEditURI, u.String())
	}
	return u.FsPath(), nil
}

func prepareRenameTarget(
	newPath string, opts *protocol.RenameFileOptions,
) (bool, error) {
	exists, err := pathExists(newPath)
	if err != nil {
		return false, err
	}
	overwrite := opts != nil && boolValue(opts.Overwrite)
	ignore := opts != nil && boolValue(opts.IgnoreIfExists)
	if exists && ignore {
		return true, nil
	}
	if exists && !overwrite {
		return false, fmt.Errorf("%w: %s", ErrWorkspaceEditFile, newPath)
	}
	if overwrite && exists {
		if err := os.RemoveAll(newPath); err != nil {
			return false, fmt.Errorf("%w: %v", ErrWorkspaceEditFile, err)
		}
	}
	return false, nil
}

func (s *Session) renameOpenDocument(oldPath, newPath string) {
	oldAbs, err := filepath.Abs(oldPath)
	if err != nil {
		return
	}
	newAbs, err := filepath.Abs(newPath)
	if err != nil {
		return
	}
	for _, doc := range s.editor.AllDocuments() {
		if doc.Path() == oldAbs {
			doc.SetPath(newAbs)
		}
	}
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("%w: %v", ErrWorkspaceEditFile, err)
}

func boolValue(v *bool) bool {
	return v != nil && *v
}

func textEditsToChanges(
	doc *view.Document, edits []protocol.TextEdit,
	encoding protocol.PositionEncodingKind,
) ([]core.Change, error) {
	changes := make([]core.Change, 0, len(edits))
	for _, edit := range edits {
		cr, ok := lspRangeToChars(doc, edit.Range, encoding)
		if !ok {
			return nil, ErrWorkspaceEditRange
		}
		changes = append(
			changes, core.TextChange(cr.From(), cr.To(), edit.NewText),
		)
	}
	slices.SortStableFunc(changes, func(a, b core.Change) int {
		if a.From != b.From {
			return cmp.Compare(a.From, b.From)
		}
		return cmp.Compare(a.To, b.To)
	})
	return changes, nil
}
