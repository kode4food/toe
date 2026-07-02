package lsp

import (
	"context"
	"unicode/utf8"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// DocumentSnapshot is the LSP-visible state of a file-backed document
type DocumentSnapshot struct {
	URI        uri.URI
	LanguageID string
	Version    int32
	Text       string
}

// SnapshotDocument captures the current file-backed view document state
func SnapshotDocument(doc *view.Document) (DocumentSnapshot, bool) {
	if doc == nil || doc.Path() == "" {
		return DocumentSnapshot{}, false
	}
	return DocumentSnapshot{
		URI:        uri.File(doc.Path()),
		LanguageID: doc.Lang(),
		Version:    int32(doc.Revision()),
		Text:       doc.Text().String(),
	}, true
}

// DidOpen sends a textDocument/didOpen notification when supported
func (c *Client) DidOpen(
	ctx context.Context, doc DocumentSnapshot,
) (bool, error) {
	if !c.openCloseEnabled() {
		return false, nil
	}
	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        doc.URI,
			LanguageID: protocol.LanguageKind(doc.LanguageID),
			Version:    doc.Version,
			Text:       doc.Text,
		},
	}
	return true, c.server.DidOpen(ctx, params)
}

// DidChange sends a textDocument/didChange notification when supported
func (c *Client) DidChange(
	ctx context.Context, doc DocumentSnapshot,
) (bool, error) {
	return c.DidChangeDocument(ctx, doc, view.DocumentChange{})
}

func (c *Client) DidChangeDocument(
	ctx context.Context, doc DocumentSnapshot, change view.DocumentChange,
) (bool, error) {
	kind, ok := c.changeSyncKind()
	if !ok {
		return false, nil
	}
	var changes []protocol.TextDocumentContentChangeEvent
	switch kind {
	case protocol.TextDocumentSyncKindFull:
		changes = []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{
				Text: doc.Text,
			},
		}
	case protocol.TextDocumentSyncKindIncremental:
		if change.Changes.Empty() {
			return false, nil
		}
		next := core.NewRope(doc.Text)
		var err error
		changes, err = incrementalChanges(
			change.Before, next, change.Changes, c.OffsetEncoding(),
		)
		if err != nil {
			return false, err
		}
	default:
		return false, nil
	}
	params := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: doc.URI,
			},
			Version: doc.Version,
		},
		ContentChanges: changes,
	}
	return true, c.server.DidChange(ctx, params)
}

// DidSave sends a textDocument/didSave notification when supported
func (c *Client) DidSave(
	ctx context.Context, doc DocumentSnapshot,
) (bool, error) {
	include, ok := c.saveEnabled()
	if !ok {
		return false, nil
	}
	params := &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: doc.URI,
		},
	}
	if include {
		params.Text = &doc.Text
	}
	return true, c.server.DidSave(ctx, params)
}

// DidClose sends a textDocument/didClose notification when supported
func (c *Client) DidClose(
	ctx context.Context, doc DocumentSnapshot,
) (bool, error) {
	if !c.openCloseEnabled() {
		return false, nil
	}
	params := &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: doc.URI,
		},
	}
	return true, c.server.DidClose(ctx, params)
}

func (c *Client) DocumentDiagnostics(
	ctx context.Context, doc DocumentSnapshot, previousID *string,
) (protocol.DocumentDiagnosticReport, bool, error) {
	identifier, ok := c.diagnosticIdentifier()
	if !ok {
		return nil, false, nil
	}
	params := &protocol.DocumentDiagnosticParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: doc.URI,
		},
		Identifier:       identifier,
		PreviousResultID: previousID,
	}
	report, err := c.server.Diagnostic(ctx, params)
	return report, true, err
}

func (c *Client) openCloseEnabled() bool {
	sync := c.textDocumentSync()
	switch v := sync.(type) {
	case protocol.TextDocumentSyncKind:
		return v != protocol.TextDocumentSyncKindNone
	case *protocol.TextDocumentSyncOptions:
		return v.OpenClose != nil && *v.OpenClose
	default:
		return false
	}
}

func (c *Client) changeSyncKind() (protocol.TextDocumentSyncKind, bool) {
	sync := c.textDocumentSync()
	switch v := sync.(type) {
	case protocol.TextDocumentSyncKind:
		return v, v != protocol.TextDocumentSyncKindNone
	case *protocol.TextDocumentSyncOptions:
		if v.Change == nil || *v.Change == protocol.TextDocumentSyncKindNone {
			return protocol.TextDocumentSyncKindNone, false
		}
		return *v.Change, true
	default:
		return protocol.TextDocumentSyncKindNone, false
	}
}

func (c *Client) saveEnabled() (bool, bool) {
	sync := c.textDocumentSync()
	if _, ok := sync.(protocol.TextDocumentSyncKind); ok {
		return false, true
	}
	opts, ok := sync.(*protocol.TextDocumentSyncOptions)
	if !ok {
		return false, false
	}
	switch v := opts.Save.(type) {
	case protocol.Boolean:
		return false, bool(v)
	case *protocol.SaveOptions:
		return v.IncludeText != nil && *v.IncludeText, true
	default:
		return false, false
	}
}

func (c *Client) textDocumentSync() protocol.TextDocumentSync {
	capabilities, ok := c.Capabilities()
	if !ok {
		return nil
	}
	return capabilities.TextDocumentSync
}

func (c *Client) diagnosticIdentifier() (*string, bool) {
	capabilities, ok := c.Capabilities()
	if !ok || capabilities.DiagnosticProvider == nil {
		return nil, false
	}
	switch p := capabilities.DiagnosticProvider.(type) {
	case *protocol.DiagnosticOptions:
		return p.Identifier, true
	case *protocol.DiagnosticRegistrationOptions:
		return p.Identifier, true
	default:
		return nil, false
	}
}

func incrementalChanges(
	before, after core.Rope, cs core.ChangeSet,
	encoding protocol.PositionEncodingKind,
) ([]protocol.TextDocumentContentChangeEvent, error) {
	ops := cs.Operations()
	changes := make([]protocol.TextDocumentContentChangeEvent, 0, len(ops))
	oldPos := 0
	newPos := 0
	for i := 0; i < len(ops); i++ {
		op := ops[i]
		oldLen := op.LenChars()
		if op.Kind() == core.OperationInsert {
			oldLen = 0
		}
		oldEnd := oldPos + oldLen
		switch op.Kind() {
		case core.OperationRetain:
			newPos += oldLen
		case core.OperationDelete:
			change, err := partialChange(
				before, after, oldPos, oldEnd, newPos, "", encoding,
			)
			if err != nil {
				return nil, err
			}
			changes = append(changes, change)
		case core.OperationInsert:
			text := op.Text()
			newPos += utf8.RuneCountInString(text)
			if i+1 < len(ops) && ops[i+1].Kind() == core.OperationDelete {
				i++
				oldEnd = oldPos + ops[i].LenChars()
			}
			change, err := partialChange(
				before, after, oldPos, oldEnd, newPos-lenRunes(text),
				text, encoding,
			)
			if err != nil {
				return nil, err
			}
			changes = append(changes, change)
		}
		oldPos = oldEnd
	}
	return changes, nil
}

func partialChange(
	before, after core.Rope, oldFrom, oldTo, newFrom int, text string,
	encoding protocol.PositionEncodingKind,
) (protocol.TextDocumentContentChangeEvent, error) {
	start, err := lspPosition(after, newFrom, encoding)
	if err != nil {
		return nil, err
	}
	oldText, err := before.SliceString(oldFrom, oldTo)
	if err != nil {
		return nil, err
	}
	end := traversePosition(start, oldText, encoding)
	return &protocol.TextDocumentContentChangePartial{
		Range: protocol.Range{
			Start: start,
			End:   end,
		},
		Text: text,
	}, nil
}

func lspPosition(
	doc core.Rope, pos int, encoding protocol.PositionEncodingKind,
) (protocol.Position, error) {
	line, err := doc.CharToLine(pos)
	if err != nil {
		return protocol.Position{}, err
	}
	lineStart, err := doc.LineToChar(line)
	if err != nil {
		return protocol.Position{}, err
	}
	text, err := doc.SliceString(lineStart, pos)
	if err != nil {
		return protocol.Position{}, err
	}
	return protocol.Position{
		Line:      uint32(line),
		Character: uint32(encodedLen(text, encoding)),
	}, nil
}

func traversePosition(
	pos protocol.Position, text string,
	encoding protocol.PositionEncodingKind,
) protocol.Position {
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\n' || ch == '\r' {
			if ch == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
				i++
			}
			pos.Line++
			pos.Character = 0
			continue
		}
		pos.Character += uint32(encodedRuneLen(ch, encoding))
	}
	return pos
}

func encodedLen(text string, encoding protocol.PositionEncodingKind) int {
	switch encoding {
	case protocol.PositionEncodingKindUTF8:
		return len(text)
	case protocol.PositionEncodingKindUTF32:
		return utf8.RuneCountInString(text)
	default:
		n := 0
		for _, ch := range text {
			n += encodedRuneLen(ch, protocol.PositionEncodingKindUTF16)
		}
		return n
	}
}

func encodedRuneLen(ch rune, encoding protocol.PositionEncodingKind) int {
	switch encoding {
	case protocol.PositionEncodingKindUTF8:
		return len(string(ch))
	case protocol.PositionEncodingKindUTF32:
		return 1
	default:
		if ch > 0xffff {
			return 2
		}
		return 1
	}
}

func lenRunes(text string) int {
	return utf8.RuneCountInString(text)
}
