package lsp

import (
	"context"
	"errors"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"go.lsp.dev/protocol"
)

// PrepareRename requests the server's rename range or placeholder
func (c *Client) PrepareRename(
	ctx context.Context, doc DocumentSnapshot, pos int,
) (protocol.PrepareRenameResult, bool, error) {
	if !c.supportsPrepareRename() {
		return nil, false, nil
	}
	return clientPosRequest(c, ctx, doc, pos, func(
		ctx context.Context, tdp protocol.TextDocumentPositionParams,
	) (protocol.PrepareRenameResult, bool, error) {
		result, err := c.server.PrepareRename(ctx, &protocol.PrepareRenameParams{
			TextDocumentPositionParams: tdp,
		})
		if err != nil {
			return nil, true, err
		}
		return result, true, nil
	})
}

// RenameSymbol requests a workspace edit for a new symbol name
func (c *Client) RenameSymbol(
	ctx context.Context, doc DocumentSnapshot, pos int, name string,
) (*protocol.WorkspaceEdit, bool, error) {
	if !c.SupportsFeature(FeatureRename) {
		return nil, false, nil
	}
	return clientPosRequest(c, ctx, doc, pos, func(
		ctx context.Context, tdp protocol.TextDocumentPositionParams,
	) (*protocol.WorkspaceEdit, bool, error) {
		edit, err := c.server.Rename(ctx, &protocol.RenameParams{
			TextDocumentPositionParams: tdp, NewName: name,
		})
		if err != nil {
			return nil, true, err
		}
		return edit, true, nil
	})
}

// RenameSymbolPrefill returns the initial rename prompt text
func (s *Session) RenameSymbolPrefill(
	doc *view.Document, viewID view.Id,
) (string, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return "", nil
	}
	pos := documentCursor(doc, viewID)
	clients := s.clientsForDocument(doc)
	if len(clients) == 0 {
		return "", ErrNoLanguageServer
	}
	supported := false
	var err error
	for _, client := range clients {
		if !client.SupportsFeature(FeatureRename) {
			continue
		}
		supported = true
		result, sent, e := client.PrepareRename(s.ctx, snap, pos)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !sent {
			continue
		}
		return renamePrefillFromResult(doc, viewID, client, result)
	}
	if !supported {
		return "", ErrNoLanguageServer
	}
	return renamePrefillFromWord(doc, viewID), err
}

// RenameSymbol applies a language-server rename edit for the current symbol
func (s *Session) RenameSymbol(
	doc *view.Document, viewID view.Id, name string,
) error {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return nil
	}
	pos := documentCursor(doc, viewID)
	clients := s.clientsForDocument(doc)
	if len(clients) == 0 {
		return ErrNoLanguageServer
	}
	var err error
	for _, client := range clients {
		edit, sent, e := client.RenameSymbol(s.ctx, snap, pos, name)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !sent {
			continue
		}
		if edit == nil {
			return err
		}
		e = s.applyWorkspaceEdit(*edit, client.OffsetEncoding())
		return errors.Join(err, e)
	}
	if err != nil {
		return err
	}
	return ErrNoLanguageServer
}

func (c *Client) supportsPrepareRename() bool {
	capabilities, ok := c.Capabilities()
	if !ok {
		return false
	}
	opts, ok := capabilities.RenameProvider.(*protocol.RenameOptions)
	return ok && opts != nil &&
		opts.PrepareProvider != nil && *opts.PrepareProvider
}

func documentCursor(doc *view.Document, viewID view.Id) int {
	sel := doc.SelectionFor(viewID)
	return sel.Primary().Cursor(doc.Text())
}

func renamePrefillFromResult(
	doc *view.Document, viewID view.Id, client *Client,
	result protocol.PrepareRenameResult,
) (string, error) {
	switch r := result.(type) {
	case *protocol.Range:
		from, to, ok := lspRangeToChars(doc, *r, client.OffsetEncoding())
		if !ok {
			return "", ErrWorkspaceEditRange
		}
		return doc.Text().SliceString(from, to)
	case *protocol.PrepareRenamePlaceholder:
		return r.Placeholder, nil
	case *protocol.PrepareRenameDefaultBehavior:
		return renamePrefillFromWord(doc, viewID), nil
	default:
		return "", nil
	}
}

func renamePrefillFromWord(doc *view.Document, viewID view.Id) string {
	sel := doc.SelectionFor(viewID)
	r := sel.Primary()
	if r.Len() > 1 {
		text, err := r.Fragment(doc.Text())
		if err == nil {
			return text
		}
	}
	r = core.TextObjectWord(doc.Text(), r, core.TextObjectInside, false)
	text, err := r.Fragment(doc.Text())
	if err != nil {
		return ""
	}
	return text
}
