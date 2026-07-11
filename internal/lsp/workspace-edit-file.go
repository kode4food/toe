package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.lsp.dev/protocol"
)

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
