package lsp

import (
	"context"
	"errors"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view"
)

var symbolKinds = [...]string{
	protocol.SymbolKindFile:          "file",
	protocol.SymbolKindModule:        "module",
	protocol.SymbolKindNamespace:     "namespace",
	protocol.SymbolKindPackage:       "package",
	protocol.SymbolKindClass:         "class",
	protocol.SymbolKindMethod:        "method",
	protocol.SymbolKindProperty:      "property",
	protocol.SymbolKindField:         "field",
	protocol.SymbolKindConstructor:   "construct",
	protocol.SymbolKindEnum:          "enum",
	protocol.SymbolKindInterface:     "interface",
	protocol.SymbolKindFunction:      "function",
	protocol.SymbolKindVariable:      "variable",
	protocol.SymbolKindConstant:      "constant",
	protocol.SymbolKindString:        "string",
	protocol.SymbolKindNumber:        "number",
	protocol.SymbolKindBoolean:       "boolean",
	protocol.SymbolKindArray:         "array",
	protocol.SymbolKindObject:        "object",
	protocol.SymbolKindKey:           "key",
	protocol.SymbolKindNull:          "null",
	protocol.SymbolKindEnumMember:    "enummem",
	protocol.SymbolKindStruct:        "struct",
	protocol.SymbolKindEvent:         "event",
	protocol.SymbolKindOperator:      "operator",
	protocol.SymbolKindTypeParameter: "typeparam",
}

// DocumentSymbols requests the document symbol tree from the server
func (c *Client) DocumentSymbols(
	ctx context.Context, doc DocumentSnapshot,
) (protocol.DocumentSymbolResult, bool, error) {
	if !c.SupportsFeature(FeatureDocumentSymbols) {
		return nil, false, nil
	}
	params := &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	result, err := c.server.DocumentSymbol(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return result, true, nil
}

// WorkspaceSymbols requests workspace symbols matching query from the server
func (c *Client) WorkspaceSymbols(
	ctx context.Context, query string,
) (protocol.WorkspaceSymbolResult, bool, error) {
	if !c.SupportsFeature(FeatureWorkspaceSymbols) {
		return nil, false, nil
	}
	params := &protocol.WorkspaceSymbolParams{Query: query}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	result, err := c.server.Symbols(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return result, true, nil
}

// DocumentSymbols returns flattened symbols from all document servers
func (s *Session) DocumentSymbols(doc *view.Document) ([]view.Symbol, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return nil, nil
	}
	var out []view.Symbol
	var err error
	for _, client := range s.clientsForDocument(doc) {
		result, sent, e := client.DocumentSymbols(s.ctx, snap)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !sent {
			continue
		}
		symbols := s.documentSymbols(client, snap, result)
		out = append(out, symbols...)
	}
	return out, err
}

// WorkspaceSymbols returns workspace symbols from all running servers
func (s *Session) WorkspaceSymbols(
	doc *view.Document, query string,
) ([]view.Symbol, error) {
	s.clientsForDocument(doc)
	var out []view.Symbol
	var err error
	for _, client := range s.servers.allClients() {
		result, sent, e := client.WorkspaceSymbols(s.ctx, query)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !sent {
			continue
		}
		symbols := s.workspaceSymbols(client, result)
		out = append(out, symbols...)
	}
	return out, err
}

func (s *Session) documentSymbols(
	client *Client, snap DocumentSnapshot, result protocol.DocumentSymbolResult,
) []view.Symbol {
	switch r := result.(type) {
	case protocol.SymbolInformationSlice:
		return s.flatSymbols(client, r)
	case protocol.DocumentSymbolSlice:
		return s.nestedSymbols(client, snap, r)
	default:
		return nil
	}
}

func (s *Session) workspaceSymbols(
	client *Client, result protocol.WorkspaceSymbolResult,
) []view.Symbol {
	switch r := result.(type) {
	case protocol.SymbolInformationSlice:
		return s.flatSymbols(client, r)
	case protocol.WorkspaceSymbolSlice:
		return s.workspaceSymbolSlice(client, r)
	default:
		return nil
	}
}

func (s *Session) flatSymbols(
	client *Client, symbols []protocol.SymbolInformation,
) []view.Symbol {
	out := make([]view.Symbol, 0, len(symbols))
	for _, sym := range symbols {
		loc, ok := s.viewLocation(client, sym.Location)
		if !ok {
			continue
		}
		container := ""
		if sym.ContainerName != nil {
			container = *sym.ContainerName
		}
		out = append(out, view.Symbol{
			Name: sym.Name, Kind: symbolKind(sym.Kind),
			Container: container, Location: loc,
		})
	}
	return out
}

func (s *Session) workspaceSymbolSlice(
	client *Client, symbols []protocol.WorkspaceSymbol,
) []view.Symbol {
	out := make([]view.Symbol, 0, len(symbols))
	for _, sym := range symbols {
		loc, ok := workspaceSymbolLocation(sym)
		if !ok {
			continue
		}
		target, ok := s.viewLocation(client, loc)
		if !ok {
			continue
		}
		container := ""
		if sym.ContainerName != nil {
			container = *sym.ContainerName
		}
		out = append(out, view.Symbol{
			Name: sym.Name, Kind: symbolKind(sym.Kind),
			Container: container, Location: target,
		})
	}
	return out
}

func (s *Session) nestedSymbols(
	client *Client, snap DocumentSnapshot, symbols []protocol.DocumentSymbol,
) []view.Symbol {
	var out []view.Symbol
	var appendSym func(container string, sym protocol.DocumentSymbol)
	appendSym = func(container string, sym protocol.DocumentSymbol) {
		loc, ok := s.viewLocation(client, protocol.Location{
			URI:   snap.URI,
			Range: sym.SelectionRange,
		})
		if ok {
			out = append(out, view.Symbol{
				Name: sym.Name, Kind: symbolKind(sym.Kind),
				Container: container, Location: loc,
			})
		}
		for _, child := range sym.Children {
			appendSym(sym.Name, child)
		}
	}
	for _, sym := range symbols {
		appendSym("", sym)
	}
	return out
}

func workspaceSymbolLocation(
	sym protocol.WorkspaceSymbol,
) (protocol.Location, bool) {
	loc, ok := sym.Location.(*protocol.Location)
	if !ok || loc == nil {
		return protocol.Location{}, false
	}
	return *loc, true
}

func symbolKind(kind protocol.SymbolKind) string {
	if int(kind) >= len(symbolKinds) {
		return ""
	}
	return symbolKinds[kind]
}
