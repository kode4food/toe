package lsp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view"
)

type (

	// CompletionList is a normalized completion response
	CompletionList struct {
		Items      []view.CompletionItem
		Raw        []protocol.CompletionItem
		Incomplete bool
	}

	completionCandidate struct {
		client *Client
		item   protocol.CompletionItem
	}
)

// Completion requests completion items from the server at the given position
func (c *Client) Completion(
	ctx context.Context, doc DocumentSnapshot, pos int,
	compCtx protocol.CompletionContext,
) (CompletionList, bool, error) {
	if !c.SupportsFeature(FeatureCompletion) {
		return CompletionList{}, false, nil
	}
	return clientPosRequest(c, posRequestArgs[CompletionList]{
		ctx: ctx,
		doc: doc,
		pos: pos,
		call: func(
			ctx context.Context, tdp protocol.TextDocumentPositionParams,
		) (CompletionList, bool, error) {
			result, err := c.server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: tdp, Context: compCtx,
			})
			if err != nil {
				return CompletionList{}, true, err
			}
			return normalizeCompletionResult(c.name, result), true, nil
		},
	})
}

// Completions requests an invoked completion list at the cursor
func (s *Session) Completions(
	doc *view.Document, viewID view.Id,
) (view.CompletionResult, error) {
	ctx := protocol.CompletionContext{
		TriggerKind: protocol.CompletionTriggerKindInvoked,
	}
	return s.completions(doc, viewID, ctx)
}

// TriggerCompletions requests character-triggered completion
func (s *Session) TriggerCompletions(
	doc *view.Document, viewID view.Id,
) (view.CompletionResult, error) {
	trigger, ok := s.completionTrigger(doc, viewID)
	if !ok {
		return view.CompletionResult{}, nil
	}
	ctx := protocol.CompletionContext{
		TriggerKind:      protocol.CompletionTriggerKindTriggerCharacter,
		TriggerCharacter: &trigger,
	}
	return s.completions(doc, viewID, ctx)
}

// ApplyCompletion applies the selected completion item to the document
func (s *Session) ApplyCompletion(
	doc *view.Document, viewID view.Id, item view.CompletionItem,
) error {
	if s.editor == nil {
		return ErrCompletionUnavailable
	}
	c, ok := s.completion(item.ID)
	if !ok {
		return ErrCompletionUnavailable
	}
	tx, err := completionTransaction(completionTransactionArgs{
		doc:      doc,
		viewID:   viewID,
		item:     c.item,
		encoding: c.client.OffsetEncoding(),
	})
	if err != nil {
		return err
	}
	if err := s.editor.Apply(tx); err != nil {
		return err
	}
	if err := s.applyAdditionalCompletionEdits(
		doc, c.item.AdditionalTextEdits, c.client.OffsetEncoding(),
	); err != nil {
		return err
	}
	return s.applyCompletionCommand(c.client, c.item.Command)
}

// ResolveCompletion fetches extra completion item details
func (s *Session) ResolveCompletion(
	_ *view.Document, _ view.Id, item view.CompletionItem,
) (view.CompletionItem, error) {
	c, ok := s.completion(item.ID)
	if !ok {
		return item, ErrCompletionUnavailable
	}
	if !clientResolvesCompletion(c.client) {
		return item, nil
	}
	ctx, cancel := c.client.requestContext(s.ctx)
	defer cancel()
	resolved, err := c.client.server.CompletionResolve(ctx, &c.item)
	if err != nil {
		return item, s.completionError(c.client, err)
	}
	if resolved == nil {
		return item, nil
	}
	c.item = *resolved
	s.storeCompletion(item.ID, c)
	out := normalizeCompletionItem(c.client.Name(), c.item)
	out.ID = item.ID
	return out, nil
}

func (s *Session) completions(
	doc *view.Document, viewID view.Id, context protocol.CompletionContext,
) (view.CompletionResult, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return view.CompletionResult{}, nil
	}
	sel := doc.SelectionFor(viewID)
	pos := sel.Primary().Cursor(doc.Text())
	clients := s.clientsForDocument(doc)
	var out []view.CompletionItem
	raw := map[string]completionCandidate{}
	var err error
	incomplete := false
	for _, client := range clients {
		list, sent, e := client.Completion(s.ctx, snap, pos, context)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if sent {
			incomplete = incomplete || list.Incomplete
			for i, item := range list.Items {
				id := candidateID(client.Name(), i)
				item.ID = id
				out = append(out, item)
				raw[id] = completionCandidate{
					client: client,
					item:   list.Raw[i],
				}
			}
		}
	}
	s.storeCompletions(raw)
	sortCompletions(out)
	return view.CompletionResult{
		Items:      out,
		Incomplete: incomplete,
	}, err
}

func (s *Session) completionError(client *Client, err error) error {
	name := client.Name()
	exited, stderr, waitErr := client.processExitedAfter(20 * time.Millisecond)
	if exited {
		s.dropClient(name, client)
		detail := stderr
		if detail == "" && waitErr != nil {
			detail = waitErr.Error()
		}
		if detail == "" {
			return fmt.Errorf("%w: %s", ErrLanguageServerExited, name)
		}
		return fmt.Errorf("%w: %s: %s", ErrLanguageServerExited, name, detail)
	}
	return fmt.Errorf("%w: %s: %w", ErrLanguageServerRequest, name, err)
}

func (s *Session) completionTrigger(
	doc *view.Document, viewID view.Id,
) (string, bool) {
	sel := doc.SelectionFor(viewID)
	pos := sel.Primary().Cursor(doc.Text())
	before, err := doc.Text().SliceString(0, pos)
	if err != nil {
		return "", false
	}
	for _, client := range s.clientsForDocument(doc) {
		capabilities, ok := client.Capabilities()
		if !ok || capabilities.CompletionProvider == nil {
			continue
		}
		provider := capabilities.CompletionProvider
		for _, trigger := range provider.TriggerCharacters {
			if trigger != "" && strings.HasSuffix(before, trigger) {
				return trigger, true
			}
		}
	}
	return "", false
}

func clientResolvesCompletion(c *Client) bool {
	capabilities, ok := c.Capabilities()
	if !ok || capabilities.CompletionProvider == nil {
		return false
	}
	return capabilities.CompletionProvider.ResolveProvider != nil &&
		*capabilities.CompletionProvider.ResolveProvider
}

func sortCompletions(items []view.CompletionItem) {
	slices.SortStableFunc(items, func(a, b view.CompletionItem) int {
		if a.Preselect != b.Preselect {
			if a.Preselect {
				return -1
			}
			return 1
		}
		if a.Sort < b.Sort {
			return -1
		}
		if a.Sort > b.Sort {
			return 1
		}
		if a.Label < b.Label {
			return -1
		}
		if a.Label > b.Label {
			return 1
		}
		return 0
	})
}

func (s *Session) storeCompletions(items map[string]completionCandidate) {
	s.candidates.Lock()
	defer s.candidates.Unlock()
	s.candidates.completions = items
}

func (s *Session) storeCompletion(id string, item completionCandidate) {
	s.candidates.Lock()
	defer s.candidates.Unlock()
	s.candidates.completions[id] = item
}

func (s *Session) completion(id string) (completionCandidate, bool) {
	s.candidates.RLock()
	defer s.candidates.RUnlock()
	c, ok := s.candidates.completions[id]
	return c, ok
}
