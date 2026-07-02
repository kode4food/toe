package lsp

import (
	"context"
	"errors"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view"
)

type (
	locationRequester func(
		*Client, context.Context, DocumentSnapshot, int,
	) ([]protocol.Location, bool, error)

	locationServerFn func(
		context.Context, protocol.TextDocumentPositionParams,
	) (any, error)
)

// GotoDeclaration requests the declaration location of the symbol at pos
func (c *Client) GotoDeclaration(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	return c.gotoLocationRequest(locationRequestArgs{
		ctx:        ctx,
		doc:        doc,
		pos:        pos,
		feature:    FeatureGotoDeclaration,
		serverCall: c.declarationLocation,
	})
}

// GotoDefinition requests the definition location of the symbol at pos
func (c *Client) GotoDefinition(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	return c.gotoLocationRequest(locationRequestArgs{
		ctx:        ctx,
		doc:        doc,
		pos:        pos,
		feature:    FeatureGotoDefinition,
		serverCall: c.definitionLocation,
	})
}

// GotoTypeDefinition requests the type definition location of the symbol at pos
func (c *Client) GotoTypeDefinition(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	return c.gotoLocationRequest(locationRequestArgs{
		ctx:        ctx,
		doc:        doc,
		pos:        pos,
		feature:    FeatureGotoTypeDefinition,
		serverCall: c.typeDefinitionLocation,
	})
}

// GotoImplementation requests implementation locations for the symbol at pos
func (c *Client) GotoImplementation(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	return c.gotoLocationRequest(locationRequestArgs{
		ctx:        ctx,
		doc:        doc,
		pos:        pos,
		feature:    FeatureGotoImplementation,
		serverCall: c.implementationLocation,
	})
}

// GotoReference requests all reference locations for the symbol at pos
func (c *Client) GotoReference(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.Location, bool, error) {
	if !c.SupportsFeature(FeatureGotoReference) {
		return nil, false, nil
	}
	return clientPosRequest(c, posRequestArgs[[]protocol.Location]{
		ctx: ctx,
		doc: doc,
		pos: pos,
		call: func(
			ctx context.Context, tdp protocol.TextDocumentPositionParams,
		) ([]protocol.Location, bool, error) {
			locations, err := c.server.References(ctx, &protocol.ReferenceParams{
				TextDocumentPositionParams: tdp,
				Context: protocol.ReferenceContext{
					IncludeDeclaration: true,
				},
			})
			if err != nil {
				return nil, true, err
			}
			return locations, true, nil
		},
	})
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

// GotoTypeDefinition returns type definition locations for the symbol at the
// cursor
func (s *Session) GotoTypeDefinition(
	doc *view.Document, viewID view.Id,
) ([]view.Location, error) {
	return s.locations(doc, viewID, (*Client).GotoTypeDefinition)
}

// GotoImplementation returns implementation locations for the symbol at the
// cursor
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

type locationRequestArgs struct {
	ctx        context.Context
	doc        DocumentSnapshot
	pos        int
	feature    Feature
	serverCall locationServerFn
}

func (c *Client) gotoLocationRequest(
	args locationRequestArgs,
) ([]protocol.Location, bool, error) {
	if !c.SupportsFeature(args.feature) {
		return nil, false, nil
	}
	return clientPosRequest(c, posRequestArgs[[]protocol.Location]{
		ctx: args.ctx,
		doc: args.doc,
		pos: args.pos,
		call: func(
			ctx context.Context, tdp protocol.TextDocumentPositionParams,
		) ([]protocol.Location, bool, error) {
			result, err := args.serverCall(ctx, tdp)
			if err != nil {
				return nil, true, err
			}
			return locationResultLocations(result), true, nil
		},
	})
}

func (c *Client) declarationLocation(
	ctx context.Context, tdp protocol.TextDocumentPositionParams,
) (any, error) {
	return c.server.Declaration(ctx, &protocol.DeclarationParams{
		TextDocumentPositionParams: tdp,
	})
}

func (c *Client) definitionLocation(
	ctx context.Context, tdp protocol.TextDocumentPositionParams,
) (any, error) {
	return c.server.Definition(ctx, &protocol.DefinitionParams{
		TextDocumentPositionParams: tdp,
	})
}

func (c *Client) typeDefinitionLocation(
	ctx context.Context, tdp protocol.TextDocumentPositionParams,
) (any, error) {
	return c.server.TypeDefinition(ctx, &protocol.TypeDefinitionParams{
		TextDocumentPositionParams: tdp,
	})
}

func (c *Client) implementationLocation(
	ctx context.Context, tdp protocol.TextDocumentPositionParams,
) (any, error) {
	return c.server.Implementation(ctx, &protocol.ImplementationParams{
		TextDocumentPositionParams: tdp,
	})
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

func locationResultLocations(result any) []protocol.Location {
	switch r := result.(type) {
	case *protocol.Location:
		if r == nil {
			return nil
		}
		return []protocol.Location{*r}
	case protocol.LocationSlice:
		return r
	case protocol.DeclarationLinkSlice:
		return linkLocations(r)
	case protocol.DefinitionLinkSlice:
		return linkLocations(r)
	default:
		return nil
	}
}

func linkLocations[T protocol.DeclarationLink | protocol.DefinitionLink](
	links []T,
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
