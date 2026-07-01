package lsp

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type fileOpSelector func(
	*protocol.FileOperationOptions,
) protocol.FileOperationRegistrationOptions

func (c *Client) WillCreateFile(
	ctx context.Context, path string, dir bool,
) (*protocol.WorkspaceEdit, bool, error) {
	if !c.fileOperationInterested(fileOpWillCreate, path, dir) {
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
	if !c.fileOperationInterested(fileOpDidCreate, path, dir) {
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
	if !c.fileOperationInterested(fileOpWillRename, oldPath, dir) {
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
	if !c.fileOperationInterested(fileOpDidRename, newPath, dir) {
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
	if !c.fileOperationInterested(fileOpWillDelete, path, dir) {
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
	if !c.fileOperationInterested(fileOpDidDelete, path, dir) {
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
		fileOpWillCreate, path, dir,
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
		fileOpDidCreate, path, dir,
	) {
		_, e := client.DidCreateFile(s.ctx, path, dir)
		err = errors.Join(err, e)
	}
	return err
}

func (s *Session) WillRenameFile(oldPath, newPath string, dir bool) error {
	var err error
	for _, client := range s.fileOperationClients(
		fileOpWillRename, oldPath, dir,
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
		fileOpDidRename, newPath, dir,
	) {
		_, e := client.DidRenameFile(s.ctx, oldPath, newPath, dir)
		err = errors.Join(err, e)
	}
	return err
}

func (s *Session) WillDeleteFile(path string, dir bool) error {
	var err error
	for _, client := range s.fileOperationClients(
		fileOpWillDelete, path, dir,
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
		fileOpDidDelete, path, dir,
	) {
		_, e := client.DidDeleteFile(s.ctx, path, dir)
		err = errors.Join(err, e)
	}
	return err
}

func (s *Session) fileOperationClients(
	sel fileOpSelector, path string, dir bool,
) []*Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Client
	for _, client := range s.clients {
		if client.fileOperationInterested(sel, path, dir) {
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
	sel fileOpSelector, path string, dir bool,
) bool {
	caps, ok := c.Capabilities()
	if !ok || caps.Workspace == nil || caps.Workspace.FileOperations == nil {
		return false
	}
	return fileOperationMatches(sel(caps.Workspace.FileOperations), path, dir)
}

func fileOpWillCreate(
	o *protocol.FileOperationOptions,
) protocol.FileOperationRegistrationOptions {
	return o.WillCreate
}

func fileOpDidCreate(
	o *protocol.FileOperationOptions,
) protocol.FileOperationRegistrationOptions {
	return o.DidCreate
}

func fileOpWillRename(
	o *protocol.FileOperationOptions,
) protocol.FileOperationRegistrationOptions {
	return o.WillRename
}

func fileOpDidRename(
	o *protocol.FileOperationOptions,
) protocol.FileOperationRegistrationOptions {
	return o.DidRename
}

func fileOpWillDelete(
	o *protocol.FileOperationOptions,
) protocol.FileOperationRegistrationOptions {
	return o.WillDelete
}

func fileOpDidDelete(
	o *protocol.FileOperationOptions,
) protocol.FileOperationRegistrationOptions {
	return o.DidDelete
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
