package lsp

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type fileOperation int

const (
	fileOperationWillCreate fileOperation = iota
	fileOperationDidCreate
	fileOperationWillRename
	fileOperationDidRename
	fileOperationWillDelete
	fileOperationDidDelete
)

func (c *Client) WillCreateFile(
	ctx context.Context, path string, dir bool,
) (*protocol.WorkspaceEdit, bool, error) {
	if !c.fileOperationInterested(fileOperationWillCreate, path, dir) {
		return nil, false, nil
	}
	params := &protocol.CreateFilesParams{
		Files: []protocol.FileCreate{{URI: fileOperationURI(path)}},
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	edit, err := c.server.WillCreateFiles(ctx, params)
	return edit, true, err
}

func (c *Client) DidCreateFile(
	ctx context.Context, path string, dir bool,
) (bool, error) {
	if !c.fileOperationInterested(fileOperationDidCreate, path, dir) {
		return false, nil
	}
	params := &protocol.CreateFilesParams{
		Files: []protocol.FileCreate{{URI: fileOperationURI(path)}},
	}
	return true, c.server.DidCreateFiles(ctx, params)
}

func (c *Client) WillRenameFile(
	ctx context.Context, oldPath, newPath string, dir bool,
) (*protocol.WorkspaceEdit, bool, error) {
	if !c.fileOperationInterested(fileOperationWillRename, oldPath, dir) {
		return nil, false, nil
	}
	params := &protocol.RenameFilesParams{
		Files: []protocol.FileRename{{
			OldURI: fileOperationURI(oldPath),
			NewURI: fileOperationURI(newPath),
		}},
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	edit, err := c.server.WillRenameFiles(ctx, params)
	return edit, true, err
}

func (c *Client) DidRenameFile(
	ctx context.Context, oldPath, newPath string, dir bool,
) (bool, error) {
	if !c.fileOperationInterested(fileOperationDidRename, newPath, dir) {
		return false, nil
	}
	params := &protocol.RenameFilesParams{
		Files: []protocol.FileRename{{
			OldURI: fileOperationURI(oldPath),
			NewURI: fileOperationURI(newPath),
		}},
	}
	return true, c.server.DidRenameFiles(ctx, params)
}

func (c *Client) WillDeleteFile(
	ctx context.Context, path string, dir bool,
) (*protocol.WorkspaceEdit, bool, error) {
	if !c.fileOperationInterested(fileOperationWillDelete, path, dir) {
		return nil, false, nil
	}
	params := &protocol.DeleteFilesParams{
		Files: []protocol.FileDelete{{URI: fileOperationURI(path)}},
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	edit, err := c.server.WillDeleteFiles(ctx, params)
	return edit, true, err
}

func (c *Client) DidDeleteFile(
	ctx context.Context, path string, dir bool,
) (bool, error) {
	if !c.fileOperationInterested(fileOperationDidDelete, path, dir) {
		return false, nil
	}
	params := &protocol.DeleteFilesParams{
		Files: []protocol.FileDelete{{URI: fileOperationURI(path)}},
	}
	return true, c.server.DidDeleteFiles(ctx, params)
}

func (s *Session) WillCreateFile(path string, dir bool) error {
	var err error
	for _, client := range s.fileOperationClients(
		fileOperationWillCreate, path, dir,
	) {
		edit, _, e := client.WillCreateFile(s.ctx, path, dir)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}
		err = errors.Join(err, s.applyFileOperationEdit(client, edit))
	}
	return err
}

func (s *Session) DidCreateFile(path string, dir bool) error {
	var err error
	for _, client := range s.fileOperationClients(
		fileOperationDidCreate, path, dir,
	) {
		_, e := client.DidCreateFile(s.ctx, path, dir)
		err = errors.Join(err, e)
	}
	return err
}

func (s *Session) WillRenameFile(oldPath, newPath string, dir bool) error {
	var err error
	for _, client := range s.fileOperationClients(
		fileOperationWillRename, oldPath, dir,
	) {
		edit, _, e := client.WillRenameFile(s.ctx, oldPath, newPath, dir)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}
		err = errors.Join(err, s.applyFileOperationEdit(client, edit))
	}
	return err
}

func (s *Session) DidRenameFile(oldPath, newPath string, dir bool) error {
	var err error
	for _, client := range s.fileOperationClients(
		fileOperationDidRename, newPath, dir,
	) {
		_, e := client.DidRenameFile(s.ctx, oldPath, newPath, dir)
		err = errors.Join(err, e)
	}
	return err
}

func (s *Session) WillDeleteFile(path string, dir bool) error {
	var err error
	for _, client := range s.fileOperationClients(
		fileOperationWillDelete, path, dir,
	) {
		edit, _, e := client.WillDeleteFile(s.ctx, path, dir)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}
		err = errors.Join(err, s.applyFileOperationEdit(client, edit))
	}
	return err
}

func (s *Session) DidDeleteFile(path string, dir bool) error {
	var err error
	for _, client := range s.fileOperationClients(
		fileOperationDidDelete, path, dir,
	) {
		_, e := client.DidDeleteFile(s.ctx, path, dir)
		err = errors.Join(err, e)
	}
	return err
}

func (s *Session) fileOperationClients(
	op fileOperation, path string, dir bool,
) []*Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []*Client{}
	for _, client := range s.clients {
		if client.fileOperationInterested(op, path, dir) {
			out = append(out, client)
		}
	}
	return out
}

func (s *Session) applyFileOperationEdit(
	client *Client, edit *protocol.WorkspaceEdit,
) error {
	if edit == nil {
		return nil
	}
	return s.applyWorkspaceEdit(*edit, client.OffsetEncoding())
}

func (c *Client) fileOperationInterested(
	op fileOperation, path string, dir bool,
) bool {
	caps, ok := c.Capabilities()
	if !ok || caps.Workspace == nil || caps.Workspace.FileOperations == nil {
		return false
	}
	opts := caps.Workspace.FileOperations
	switch op {
	case fileOperationWillCreate:
		return fileOperationMatches(opts.WillCreate, path, dir)
	case fileOperationDidCreate:
		return fileOperationMatches(opts.DidCreate, path, dir)
	case fileOperationWillRename:
		return fileOperationMatches(opts.WillRename, path, dir)
	case fileOperationDidRename:
		return fileOperationMatches(opts.DidRename, path, dir)
	case fileOperationWillDelete:
		return fileOperationMatches(opts.WillDelete, path, dir)
	case fileOperationDidDelete:
		return fileOperationMatches(opts.DidDelete, path, dir)
	default:
		return false
	}
}

func fileOperationMatches(
	opts protocol.FileOperationRegistrationOptions, path string, dir bool,
) bool {
	for _, filter := range opts.Filters {
		if !fileOperationSchemeOK(filter.Scheme) {
			continue
		}
		if !fileOperationKindOK(filter.Pattern.Matches, dir) {
			continue
		}
		pattern := filepath.FromSlash(filter.Pattern.Glob)
		candidate := path
		if ignoreCase(filter.Pattern.Options) {
			pattern = strings.ToLower(pattern)
			candidate = strings.ToLower(candidate)
		}
		if matchWatchPattern(pattern, candidate) {
			return true
		}
	}
	return false
}

func fileOperationSchemeOK(scheme *string) bool {
	return scheme == nil || *scheme == "file"
}

func fileOperationKindOK(
	kind protocol.FileOperationPatternKind, dir bool,
) bool {
	switch kind {
	case protocol.FileOperationPatternKindFile:
		return !dir
	case protocol.FileOperationPatternKindFolder:
		return dir
	default:
		return true
	}
}

func ignoreCase(opts *protocol.FileOperationPatternOptions) bool {
	return opts != nil && opts.IgnoreCase != nil && *opts.IgnoreCase
}

func fileOperationURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return string(uri.File(path))
	}
	return string(uri.File(abs))
}
