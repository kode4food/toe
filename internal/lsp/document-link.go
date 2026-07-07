package lsp

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view"
)

type documentLinkCandidate struct {
	client *Client
	docID  view.DocumentId
	link   protocol.DocumentLink
}

// DocumentLinks requests document-wide links
func (c *Client) DocumentLinks(
	ctx context.Context, doc DocumentSnapshot,
) ([]protocol.DocumentLink, bool, error) {
	return clientDocRequest(c, ctx, doc,
		func(
			ctx context.Context, c *Client,
			tdid protocol.TextDocumentIdentifier,
		) ([]protocol.DocumentLink, bool, error) {
			if !c.SupportsFeature(FeatureDocumentLinks) {
				return nil, false, nil
			}
			links, err := c.server.DocumentLink(
				ctx, &protocol.DocumentLinkParams{TextDocument: tdid},
			)
			if err != nil {
				return nil, true, err
			}
			return links, true, nil
		},
	)
}

// ResolveDocumentLink resolves a stored document link target when supported
func (c *Client) ResolveDocumentLink(
	ctx context.Context, link protocol.DocumentLink,
) (*protocol.DocumentLink, bool, error) {
	if !clientResolvesDocumentLinks(c) {
		return nil, false, nil
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	resolved, err := c.server.DocumentLinkResolve(ctx, &link)
	if err != nil {
		return nil, true, err
	}
	return resolved, true, nil
}

// DocumentLinks returns document-wide links and stores them on the document
func (s *Session) DocumentLinks(
	doc *view.Document,
) ([]view.DocumentLink, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		doc.ClearDocumentLinks()
		return nil, nil
	}
	var out []view.DocumentLink
	raw := map[string]documentLinkCandidate{}
	var sent bool
	var err error
	for _, client := range s.clientsForDocument(doc) {
		links, ok, e := client.DocumentLinks(s.ctx, snap)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !ok {
			continue
		}
		sent = true
		for i, link := range links {
			item, ok := viewDocumentLink(client, doc, i, link)
			if !ok {
				continue
			}
			raw[item.ID] = documentLinkCandidate{
				client: client, docID: doc.ID(), link: link,
			}
			out = append(out, item)
		}
	}
	if !sent {
		doc.ClearDocumentLinks()
		return nil, ErrNoLanguageServer
	}
	slices.SortFunc(out, func(a, b view.DocumentLink) int {
		if n := cmp.Compare(a.From, b.From); n != 0 {
			return n
		}
		return cmp.Compare(a.To, b.To)
	})
	doc.SetDocumentLinks(out)
	s.storeDocumentLinks(doc.ID(), raw)
	return out, err
}

// ResolveDocumentLink resolves a document link target and updates the document
func (s *Session) ResolveDocumentLink(
	doc *view.Document, link view.DocumentLink,
) (view.DocumentLink, error) {
	if link.Target != "" {
		return link, nil
	}
	candidate, ok := s.documentLink(link.ID)
	if !ok || candidate.docID != doc.ID() {
		return link, ErrDocumentLinkUnavailable
	}
	resolved, sent, err := candidate.client.ResolveDocumentLink(
		s.ctx, candidate.link,
	)
	if err != nil {
		return link, s.completionError(candidate.client, err)
	}
	if !sent || resolved == nil || resolved.Target == nil {
		return link, nil
	}
	item, ok := viewDocumentLink(candidate.client, doc, 0, *resolved)
	if !ok {
		return link, nil
	}
	item.ID = link.ID
	s.replaceDocumentLink(doc, link.ID, item, *resolved)
	return item, nil
}

func (s *Session) storeDocumentLinks(
	docID view.DocumentId, links map[string]documentLinkCandidate,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearDocumentLinksLocked(docID)
	maps.Copy(s.links, links)
}

func (s *Session) clearDocumentLinksLocked(docID view.DocumentId) {
	for id, link := range s.links {
		if link.docID == docID {
			delete(s.links, id)
		}
	}
}

func (s *Session) documentLink(id string) (documentLinkCandidate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.links[id]
	return link, ok
}

func (s *Session) replaceDocumentLink(
	doc *view.Document, id string, link view.DocumentLink,
	raw protocol.DocumentLink,
) {
	links := doc.DocumentLinks()
	for i := range links {
		if links[i].ID == id {
			links[i] = link
			break
		}
	}
	doc.SetDocumentLinks(links)

	s.mu.Lock()
	if candidate, ok := s.links[id]; ok {
		candidate.link = raw
		s.links[id] = candidate
	}
	s.mu.Unlock()
}

func (s *Session) documentLinksAsync(doc *view.Document) {
	go func() {
		_, _ = s.DocumentLinks(doc)
	}()
}

func viewDocumentLink(
	client *Client, doc *view.Document, idx int, link protocol.DocumentLink,
) (view.DocumentLink, bool) {
	cr, ok := lspRangeToChars(doc, link.Range, client.OffsetEncoding())
	if !ok || cr.From() >= cr.To() {
		return view.DocumentLink{}, false
	}
	target := ""
	if link.Target != nil {
		target = link.Target.String()
	}
	return view.DocumentLink{
		ID:     candidateID(client.Name(), idx),
		From:   cr.From(),
		To:     cr.To(),
		Target: target,
		Server: client.Name(),
	}, true
}

func clientResolvesDocumentLinks(client *Client) bool {
	capabilities, ok := client.Capabilities()
	if !ok || capabilities.DocumentLinkProvider == nil {
		return false
	}
	resolve := capabilities.DocumentLinkProvider.ResolveProvider
	return resolve != nil && *resolve
}
