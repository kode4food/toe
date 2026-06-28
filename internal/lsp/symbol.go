package lsp

import (
	"context"
	"errors"

	"github.com/kode4food/toe/internal/view"
	"go.lsp.dev/protocol"
)

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

// DocumentSymbols returns the flattened symbol list for the document from all servers
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

func (s *Session) nestedSymbols(
	client *Client, snap DocumentSnapshot, symbols []protocol.DocumentSymbol,
) []view.Symbol {
	var out []view.Symbol
	for _, sym := range symbols {
		s.appendNestedSymbol(client, snap, &out, sym, "")
	}
	return out
}

func (s *Session) appendNestedSymbol(
	client *Client, snap DocumentSnapshot, out *[]view.Symbol,
	sym protocol.DocumentSymbol, container string,
) {
	loc, ok := s.viewLocation(client, protocol.Location{
		URI:   snap.URI,
		Range: sym.SelectionRange,
	})
	if ok {
		*out = append(*out, view.Symbol{
			Name: sym.Name, Kind: symbolKind(sym.Kind),
			Container: container, Location: loc,
		})
	}
	for _, child := range sym.Children {
		s.appendNestedSymbol(client, snap, out, child, sym.Name)
	}
}

func symbolKind(kind protocol.SymbolKind) string {
	switch kind {
	case protocol.SymbolKindFile:
		return "file"
	case protocol.SymbolKindModule:
		return "module"
	case protocol.SymbolKindNamespace:
		return "namespace"
	case protocol.SymbolKindPackage:
		return "package"
	case protocol.SymbolKindClass:
		return "class"
	case protocol.SymbolKindMethod:
		return "method"
	case protocol.SymbolKindProperty:
		return "property"
	case protocol.SymbolKindField:
		return "field"
	case protocol.SymbolKindConstructor:
		return "construct"
	case protocol.SymbolKindEnum:
		return "enum"
	case protocol.SymbolKindInterface:
		return "interface"
	case protocol.SymbolKindFunction:
		return "function"
	case protocol.SymbolKindVariable:
		return "variable"
	case protocol.SymbolKindConstant:
		return "constant"
	case protocol.SymbolKindString:
		return "string"
	case protocol.SymbolKindNumber:
		return "number"
	case protocol.SymbolKindBoolean:
		return "boolean"
	case protocol.SymbolKindArray:
		return "array"
	case protocol.SymbolKindObject:
		return "object"
	case protocol.SymbolKindKey:
		return "key"
	case protocol.SymbolKindNull:
		return "null"
	case protocol.SymbolKindEnumMember:
		return "enummem"
	case protocol.SymbolKindStruct:
		return "struct"
	case protocol.SymbolKindEvent:
		return "event"
	case protocol.SymbolKindOperator:
		return "operator"
	case protocol.SymbolKindTypeParameter:
		return "typeparam"
	default:
		return ""
	}
}
