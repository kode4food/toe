package lsp

import (
	"context"
	"errors"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"go.lsp.dev/protocol"
)

type locationRequester func(
	*Client, context.Context, DocumentSnapshot, int,
) ([]protocol.Location, bool, error)

// GotoDeclaration requests the declaration location of the symbol at pos
func (c *Client) GotoDeclaration(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	if !c.SupportsFeature(FeatureGotoDeclaration) {
		return nil, false, nil
	}
	lspPos, err := lspPosition(
		core.NewRope(doc.Text), pos, c.OffsetEncoding(),
	)
	if err != nil {
		return nil, false, err
	}
	params := &protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
			Position:     lspPos,
		},
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	result, err := c.server.Declaration(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return declarationResultLocations(result), true, nil
}

// GotoDefinition requests the definition location of the symbol at pos
func (c *Client) GotoDefinition(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	return c.definitionRequest(ctx, doc, pos, FeatureGotoDefinition, definition)
}

// GotoTypeDefinition requests the type definition location of the symbol at pos
func (c *Client) GotoTypeDefinition(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	return c.definitionRequest(
		ctx, doc, pos, FeatureGotoTypeDefinition, typeDefinition,
	)
}

// GotoImplementation requests the implementation locations for the symbol at pos
func (c *Client) GotoImplementation(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	return c.definitionRequest(
		ctx, doc, pos, FeatureGotoImplementation, implementation,
	)
}

// GotoReference requests all reference locations for the symbol at pos
func (c *Client) GotoReference(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	if !c.SupportsFeature(FeatureGotoReference) {
		return nil, false, nil
	}
	lspPos, err := lspPosition(
		core.NewRope(doc.Text), pos, c.OffsetEncoding(),
	)
	if err != nil {
		return nil, false, err
	}
	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
			Position:     lspPos,
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	locations, err := c.server.References(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return locations, true, nil
}

// GotoDeclaration returns declaration locations for the symbol at the cursor
func (s *Session) GotoDeclaration(
	doc *view.Document, viewID view.Id,
) ([]view.Location, error) {
	return s.locations(doc, viewID, (*Client).GotoDeclaration)
}

// GotoDefinition returns definition locations for the symbol at the cursor
func (s *Session) GotoDefinition(
	doc *view.Document, viewID view.Id,
) ([]view.Location, error) {
	return s.locations(doc, viewID, (*Client).GotoDefinition)
}

// GotoTypeDefinition returns type definition locations for the symbol at the cursor
func (s *Session) GotoTypeDefinition(
	doc *view.Document, viewID view.Id,
) ([]view.Location, error) {
	return s.locations(doc, viewID, (*Client).GotoTypeDefinition)
}

// GotoImplementation returns implementation locations for the symbol at the cursor
func (s *Session) GotoImplementation(
	doc *view.Document, viewID view.Id,
) ([]view.Location, error) {
	return s.locations(doc, viewID, (*Client).GotoImplementation)
}

// GotoReference returns all reference locations for the symbol at the cursor
func (s *Session) GotoReference(
	doc *view.Document, viewID view.Id,
) ([]view.Location, error) {
	return s.locations(doc, viewID, (*Client).GotoReference)
}

func (c *Client) definitionRequest(
	ctx context.Context, doc DocumentSnapshot, pos int, feature Feature,
	request func(
		context.Context, protocol.Server, *protocol.TextDocumentPositionParams,
	) (protocol.DefinitionResult, error),
) ([]protocol.Location, bool, error) {
	if !c.SupportsFeature(feature) {
		return nil, false, nil
	}
	lspPos, err := lspPosition(
		core.NewRope(doc.Text), pos, c.OffsetEncoding(),
	)
	if err != nil {
		return nil, false, err
	}
	params := &protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
		Position:     lspPos,
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	result, err := request(ctx, c.server, params)
	if err != nil {
		return nil, true, err
	}
	return definitionResultLocations(result), true, nil
}

func (s *Session) locations(
	doc *view.Document, viewID view.Id, request locationRequester,
) ([]view.Location, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return nil, nil
	}
	sel := doc.SelectionFor(viewID)
	pos := sel.Primary().Cursor(doc.Text())
	var out []view.Location
	var err error
	for _, client := range s.clientsForDocument(doc) {
		locations, sent, e := request(client, s.ctx, snap, pos)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !sent || len(locations) == 0 {
			continue
		}
		converted := s.viewLocations(client, locations)
		out = append(out, converted...)
	}
	return out, err
}

func (s *Session) viewLocations(
	client *Client, locations []protocol.Location,
) []view.Location {
	out := make([]view.Location, 0, len(locations))
	for _, loc := range locations {
		if target, ok := s.viewLocation(client, loc); ok {
			out = append(out, target)
		}
	}
	return out
}

func (s *Session) viewLocation(
	client *Client, loc protocol.Location,
) (view.Location, bool) {
	if !loc.URI.IsFile() || s.editor == nil {
		return view.Location{}, false
	}
	doc, err := s.editor.SwitchOrOpenDoc(loc.URI.FsPath())
	if err != nil {
		return view.Location{}, false
	}
	encoding := client.OffsetEncoding()
	from, ok := lspPositionToChar(doc, loc.Range.Start, encoding)
	if !ok {
		return view.Location{}, false
	}
	to, ok := lspPositionToChar(doc, loc.Range.End, encoding)
	if !ok {
		to = from
	}
	return view.Location{Path: doc.Path(), From: from, To: to}, true
}

func definition(
	ctx context.Context, server protocol.Server,
	params *protocol.TextDocumentPositionParams,
) (protocol.DefinitionResult, error) {
	return server.Definition(ctx, &protocol.DefinitionParams{
		TextDocumentPositionParams: *params,
	})
}

func typeDefinition(
	ctx context.Context, server protocol.Server,
	params *protocol.TextDocumentPositionParams,
) (protocol.DefinitionResult, error) {
	return server.TypeDefinition(ctx, &protocol.TypeDefinitionParams{
		TextDocumentPositionParams: *params,
	})
}

func implementation(
	ctx context.Context, server protocol.Server,
	params *protocol.TextDocumentPositionParams,
) (protocol.DefinitionResult, error) {
	return server.Implementation(ctx, &protocol.ImplementationParams{
		TextDocumentPositionParams: *params,
	})
}

func declarationResultLocations(
	result protocol.DeclarationResult,
) []protocol.Location {
	switch r := result.(type) {
	case *protocol.Location:
		if r == nil {
			return nil
		}
		return []protocol.Location{*r}
	case protocol.LocationSlice:
		return []protocol.Location(r)
	case protocol.DeclarationLinkSlice:
		return declarationLinkLocations(r)
	default:
		return nil
	}
}

func definitionResultLocations(
	result protocol.DefinitionResult,
) []protocol.Location {
	switch r := result.(type) {
	case *protocol.Location:
		if r == nil {
			return nil
		}
		return []protocol.Location{*r}
	case protocol.LocationSlice:
		return []protocol.Location(r)
	case protocol.DefinitionLinkSlice:
		return definitionLinkLocations(r)
	default:
		return nil
	}
}

func declarationLinkLocations(
	links protocol.DeclarationLinkSlice,
) []protocol.Location {
	out := make([]protocol.Location, 0, len(links))
	for _, link := range links {
		target := protocol.LocationLink(link)
		out = append(out, protocol.Location{
			URI:   target.TargetURI,
			Range: target.TargetSelectionRange,
		})
	}
	return out
}

func definitionLinkLocations(
	links protocol.DefinitionLinkSlice,
) []protocol.Location {
	out := make([]protocol.Location, 0, len(links))
	for _, link := range links {
		target := protocol.LocationLink(link)
		out = append(out, protocol.Location{
			URI:   target.TargetURI,
			Range: target.TargetSelectionRange,
		})
	}
	return out
}
