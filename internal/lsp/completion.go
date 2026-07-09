package lsp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
	completionCandidate struct {
		client *Client
		item   protocol.CompletionItem
	}

	completionEditOffset struct {
		from int
		to   int
	}

	completionEditCtx struct {
		doc      *view.Document
		encoding protocol.PositionEncodingKind
		cursor   int
	}

	completionApplyOp struct {
		offset  *completionEditOffset
		removed string
		newText string
	}

	// CompletionList is a normalized completion response
	CompletionList struct {
		Items      []view.CompletionItem
		Raw        []protocol.CompletionItem
		Incomplete bool
	}
)

var completionItemKindNames = map[protocol.CompletionItemKind]string{
	protocol.CompletionItemKindText:          "text",
	protocol.CompletionItemKindMethod:        "method",
	protocol.CompletionItemKindFunction:      "function",
	protocol.CompletionItemKindConstructor:   "constructor",
	protocol.CompletionItemKindField:         "field",
	protocol.CompletionItemKindVariable:      "variable",
	protocol.CompletionItemKindClass:         "class",
	protocol.CompletionItemKindInterface:     "interface",
	protocol.CompletionItemKindModule:        "module",
	protocol.CompletionItemKindProperty:      "property",
	protocol.CompletionItemKindUnit:          "unit",
	protocol.CompletionItemKindValue:         "value",
	protocol.CompletionItemKindEnum:          "enum",
	protocol.CompletionItemKindKeyword:       "keyword",
	protocol.CompletionItemKindSnippet:       "snippet",
	protocol.CompletionItemKindColor:         "color",
	protocol.CompletionItemKindFile:          "file",
	protocol.CompletionItemKindReference:     "reference",
	protocol.CompletionItemKindFolder:        "folder",
	protocol.CompletionItemKindEnumMember:    "enum_member",
	protocol.CompletionItemKindConstant:      "constant",
	protocol.CompletionItemKindStruct:        "struct",
	protocol.CompletionItemKindEvent:         "event",
	protocol.CompletionItemKindOperator:      "operator",
	protocol.CompletionItemKindTypeParameter: "type_param",
}

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

func normalizeCompletionResult(
	server string, result protocol.CompletionResult,
) CompletionList {
	switch r := result.(type) {
	case protocol.CompletionItemSlice:
		return CompletionList{
			Items: normalizeCompletionItems(server, r), Raw: r,
		}
	case *protocol.CompletionList:
		return CompletionList{
			Items:      normalizeCompletionItems(server, r.Items),
			Raw:        r.Items,
			Incomplete: r.IsIncomplete,
		}
	default:
		return CompletionList{}
	}
}

func normalizeCompletionItems(
	server string, items []protocol.CompletionItem,
) []view.CompletionItem {
	out := make([]view.CompletionItem, 0, len(items))
	for _, item := range items {
		out = append(out, normalizeCompletionItem(server, item))
	}
	return out
}

func normalizeCompletionItem(
	server string, item protocol.CompletionItem,
) view.CompletionItem {
	filter := item.Label
	if text, ok := item.FilterText.Get(); ok {
		filter = text
	}
	sortText := item.Label
	if text, ok := item.SortText.Get(); ok {
		sortText = text
	}
	insert := item.Label
	if text, ok := item.InsertText.Get(); ok {
		insert = text
	}
	detail, _ := item.Detail.Get()
	preselect, _ := item.Preselect.Get()
	deprecated := completionDeprecated(item.Tags)
	labelDetail, labelDescription := completionLabelDetails(item)
	return view.CompletionItem{
		Label:            item.Label,
		LabelDetail:      labelDetail,
		LabelDescription: labelDescription,
		Detail:           detail,
		Filter:           filter,
		Sort:             sortText,
		Insert:           insert,
		Kind:             completionItemKind(item.Kind),
		Docs:             completionDocumentation(detail, item.Documentation),
		Server:           server,
		Preselect:        preselect,
		Deprecated:       deprecated,
	}
}

func completionLabelDetails(item protocol.CompletionItem) (string, string) {
	if item.LabelDetails == nil {
		return "", ""
	}
	detail := ""
	if item.LabelDetails.Detail != nil {
		detail = *item.LabelDetails.Detail
	}
	description := ""
	if item.LabelDetails.Description != nil {
		description = *item.LabelDetails.Description
	}
	return detail, description
}

func completionDeprecated(tags []protocol.CompletionItemTag) bool {
	return slices.Contains(tags, protocol.CompletionItemTagDeprecated)
}

func completionItemKind(kind protocol.CompletionItemKind) string {
	return completionItemKindNames[kind]
}

func completionDocumentation(
	detail string, docs protocol.InlayHintTooltip,
) string {
	doc := markupText(docs)
	switch {
	case detail != "" && doc != "":
		return "```text\n" + detail + "\n```\n" + doc
	case detail != "":
		return "```text\n" + detail + "\n```"
	default:
		return doc
	}
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

func (s *Session) applyAdditionalCompletionEdits(
	doc *view.Document, edits []protocol.TextEdit,
	encoding protocol.PositionEncodingKind,
) error {
	if len(edits) == 0 {
		return nil
	}
	changes := make([]core.Change, 0, len(edits))
	for _, edit := range edits {
		cr, ok := lspRangeToChars(doc, edit.Range, encoding)
		if !ok {
			return ErrCompletionUnavailable
		}
		changes = append(changes, core.TextChange(cr.From(), cr.To(), edit.NewText))
	}
	cs, err := core.NewChangeSetFromChanges(doc.Text(), changes)
	if err != nil {
		return err
	}
	tx := core.NewTransaction(doc.Text()).WithChanges(cs)
	return s.editor.Apply(tx)
}

func (s *Session) applyCompletionCommand(
	client *Client, command protocol.Command,
) error {
	if command.Command == "" {
		return nil
	}
	params := &protocol.ExecuteCommandParams{
		Command:   command.Command,
		Arguments: command.Arguments,
	}
	return client.ExecuteCommand(s.ctx, params)
}

type completionTransactionArgs struct {
	doc      *view.Document
	item     protocol.CompletionItem
	encoding protocol.PositionEncodingKind
	viewID   view.Id
}

func completionTransaction(
	args completionTransactionArgs,
) (core.Transaction, error) {
	doc := args.doc
	sel := doc.SelectionFor(args.viewID)
	text := doc.Text()
	cursor := sel.Primary().Cursor(text)
	ctx := completionEditCtx{doc: doc, encoding: args.encoding, cursor: cursor}
	editOffset, newText, err := completionEdit(ctx, args.item)
	if err != nil {
		return core.Transaction{}, err
	}
	from, to, ok := completionRange(text, editOffset, cursor)
	if !ok {
		return core.Transaction{}, ErrCompletionUnavailable
	}
	from, to = completionPrimaryRange(text, from, to, cursor)
	removed, err := text.SliceString(from, to)
	if err != nil {
		return core.Transaction{}, err
	}
	apply := completionApplyOp{
		offset: editOffset, removed: removed, newText: newText,
	}
	changes, err := completionChanges(text, sel, apply)
	if err != nil {
		return core.Transaction{}, err
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return core.Transaction{}, err
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return core.Transaction{}, err
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	return tx, nil
}

func completionPrimaryRange(text core.Rope, from, to, cursor int) (int, int) {
	wordFrom, wordTo := findCompletionRange(text, cursor)
	if from <= wordFrom && to >= wordTo {
		return from, to
	}
	return wordFrom, wordTo
}

func completionEdit(
	ctx completionEditCtx, item protocol.CompletionItem,
) (*completionEditOffset, string, error) {
	switch edit := item.TextEdit.(type) {
	case *protocol.TextEdit:
		cr, ok := lspRangeToChars(ctx.doc, edit.Range, ctx.encoding)
		if !ok {
			return nil, "", ErrCompletionUnavailable
		}
		return &completionEditOffset{
			from: cr.From() - ctx.cursor,
			to:   cr.To() - ctx.cursor,
		}, edit.NewText, nil
	case *protocol.InsertReplaceEdit:
		cr, ok := lspRangeToChars(ctx.doc, edit.Insert, ctx.encoding)
		if !ok {
			return nil, "", ErrCompletionUnavailable
		}
		return &completionEditOffset{
			from: cr.From() - ctx.cursor,
			to:   cr.To() - ctx.cursor,
		}, edit.NewText, nil
	default:
		if text, ok := item.InsertText.Get(); ok {
			return nil, text, nil
		}
		return nil, item.Label, nil
	}
}

func completionChanges(
	text core.Rope, sel core.Selection, op completionApplyOp,
) ([]core.Change, error) {
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		cursor := r.Cursor(text)
		from, to := completionRangeForCursor(text, op.offset, cursor)
		got, err := text.SliceString(from, to)
		if err != nil {
			return nil, err
		}
		if got != op.removed {
			from, to = findCompletionRange(text, cursor)
		}
		changes = append(changes, core.TextChange(from, to, op.newText))
	}
	return changes, nil
}

func completionRangeForCursor(
	text core.Rope, offset *completionEditOffset, cursor int,
) (int, int) {
	if from, to, ok := completionRange(text, offset, cursor); ok {
		return from, to
	}
	return findCompletionRange(text, cursor)
}

func completionRange(
	text core.Rope, offset *completionEditOffset, cursor int,
) (int, int, bool) {
	if offset == nil {
		from, to := findCompletionRange(text, cursor)
		return from, to, true
	}
	from := cursor + offset.from
	to := cursor + offset.to
	if from < 0 || to > text.LenChars() || from > to {
		return 0, 0, false
	}
	return from, to, true
}

func findCompletionRange(text core.Rope, cursor int) (int, int) {
	before, _ := text.SliceString(0, cursor)
	from := cursor - countWordSuffix(before)
	return from, cursor
}

func countWordSuffix(s string) int {
	runes := []rune(s)
	n := 0
	for i := len(runes) - 1; i >= 0; i-- {
		if !core.CharIsWord(runes[i]) {
			return n
		}
		n++
	}
	return n
}
