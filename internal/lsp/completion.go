package lsp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"go.lsp.dev/protocol"
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

	// CompletionList is a normalized completion response
	CompletionList struct {
		Items      []view.CompletionItem
		Raw        []protocol.CompletionItem
		Incomplete bool
	}
)

// Completion requests completion items from the server at the given position
func (c *Client) Completion(
	ctx context.Context, doc DocumentSnapshot, pos int,
	context protocol.CompletionContext,
) (CompletionList, bool, error) {
	if !c.SupportsFeature(FeatureCompletion) {
		return CompletionList{}, false, nil
	}
	lspPos, err := lspPosition(
		core.NewRope(doc.Text), pos, c.OffsetEncoding(),
	)
	if err != nil {
		return CompletionList{}, false, err
	}
	params := &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
			Position:     lspPos,
		},
		Context: context,
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	result, err := c.server.Completion(ctx, params)
	if err != nil {
		return CompletionList{}, true, err
	}
	return normalizeCompletionResult(c.name, result), true, nil
}

// Completions requests an invoked completion list at the cursor
func (s *Session) Completions(
	doc *view.Document, viewID view.Id,
) (view.CompletionResult, error) {
	context := protocol.CompletionContext{
		TriggerKind: protocol.CompletionTriggerKindInvoked,
	}
	return s.completions(doc, viewID, context)
}

// TriggerCompletions requests character-triggered completion
func (s *Session) TriggerCompletions(
	doc *view.Document, viewID view.Id,
) (view.CompletionResult, error) {
	trigger, ok := s.completionTrigger(doc, viewID)
	if !ok {
		return view.CompletionResult{}, nil
	}
	context := protocol.CompletionContext{
		TriggerKind:      protocol.CompletionTriggerKindTriggerCharacter,
		TriggerCharacter: &trigger,
	}
	return s.completions(doc, viewID, context)
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
	out := []view.CompletionItem{}
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
				id := completionID(client.Name(), i)
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
	return fmt.Errorf("%w: %s", ErrLanguageServerRequest, name)
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
	switch kind {
	case protocol.CompletionItemKindText:
		return "text"
	case protocol.CompletionItemKindMethod:
		return "method"
	case protocol.CompletionItemKindFunction:
		return "function"
	case protocol.CompletionItemKindConstructor:
		return "constructor"
	case protocol.CompletionItemKindField:
		return "field"
	case protocol.CompletionItemKindVariable:
		return "variable"
	case protocol.CompletionItemKindClass:
		return "class"
	case protocol.CompletionItemKindInterface:
		return "interface"
	case protocol.CompletionItemKindModule:
		return "module"
	case protocol.CompletionItemKindProperty:
		return "property"
	case protocol.CompletionItemKindUnit:
		return "unit"
	case protocol.CompletionItemKindValue:
		return "value"
	case protocol.CompletionItemKindEnum:
		return "enum"
	case protocol.CompletionItemKindKeyword:
		return "keyword"
	case protocol.CompletionItemKindSnippet:
		return "snippet"
	case protocol.CompletionItemKindColor:
		return "color"
	case protocol.CompletionItemKindFile:
		return "file"
	case protocol.CompletionItemKindReference:
		return "reference"
	case protocol.CompletionItemKindFolder:
		return "folder"
	case protocol.CompletionItemKindEnumMember:
		return "enum_member"
	case protocol.CompletionItemKindConstant:
		return "constant"
	case protocol.CompletionItemKindStruct:
		return "struct"
	case protocol.CompletionItemKindEvent:
		return "event"
	case protocol.CompletionItemKindOperator:
		return "operator"
	case protocol.CompletionItemKindTypeParameter:
		return "type_param"
	default:
		return ""
	}
}

func completionDocumentation(
	detail string, docs protocol.InlayHintTooltip,
) string {
	doc := tooltipText(docs)
	switch {
	case detail != "" && doc != "":
		return "```text\n" + detail + "\n```\n" + doc
	case detail != "":
		return "```text\n" + detail + "\n```"
	default:
		return doc
	}
}

func tooltipText(docs protocol.InlayHintTooltip) string {
	switch v := docs.(type) {
	case protocol.String:
		return string(v)
	case *protocol.MarkupContent:
		if v == nil {
			return ""
		}
		return v.Value
	default:
		return ""
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

func completionID(server string, idx int) string {
	return server + ":" + strconv.Itoa(idx)
}

func (s *Session) storeCompletions(items map[string]completionCandidate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.comps = items
}

func (s *Session) storeCompletion(id string, item completionCandidate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.comps[id] = item
}

func (s *Session) completion(id string) (completionCandidate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.comps[id]
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
		from, to, ok := completionEditRange(doc, edit.Range, encoding)
		if !ok {
			return ErrCompletionUnavailable
		}
		changes = append(changes, core.TextChange(from, to, edit.NewText))
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
	replace  bool
}

func completionTransaction(
	args completionTransactionArgs,
) (core.Transaction, error) {
	doc := args.doc
	sel := doc.SelectionFor(args.viewID)
	text := doc.Text()
	cursor := sel.Primary().Cursor(text)
	editOffset, newText, err := completionEdit(
		doc, args.item, args.encoding, cursor, args.replace,
	)
	if err != nil {
		return core.Transaction{}, err
	}
	from, to, ok := completionRange(text, editOffset, args.replace, cursor)
	if !ok {
		return core.Transaction{}, ErrCompletionUnavailable
	}
	from, to = completionPrimaryRange(text, from, to, args.replace, cursor)
	removed, err := text.SliceString(from, to)
	if err != nil {
		return core.Transaction{}, err
	}
	changes, err := completionChanges(
		text, sel, editOffset, args.replace, removed, newText,
	)
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

func completionPrimaryRange(
	text core.Rope, from, to int, replace bool, cursor int,
) (int, int) {
	wordFrom, wordTo := findCompletionRange(text, replace, cursor)
	if from <= wordFrom && to >= wordTo {
		return from, to
	}
	return wordFrom, wordTo
}

func completionEdit(
	doc *view.Document, item protocol.CompletionItem,
	encoding protocol.PositionEncodingKind, cursor int, replace bool,
) (*completionEditOffset, string, error) {
	switch edit := item.TextEdit.(type) {
	case *protocol.TextEdit:
		from, to, ok := completionEditRange(doc, edit.Range, encoding)
		if !ok {
			return nil, "", ErrCompletionUnavailable
		}
		return &completionEditOffset{
			from: from - cursor,
			to:   to - cursor,
		}, edit.NewText, nil
	case *protocol.InsertReplaceEdit:
		r := edit.Insert
		if replace {
			r = edit.Replace
		}
		from, to, ok := completionEditRange(doc, r, encoding)
		if !ok {
			return nil, "", ErrCompletionUnavailable
		}
		return &completionEditOffset{
			from: from - cursor,
			to:   to - cursor,
		}, edit.NewText, nil
	default:
		if text, ok := item.InsertText.Get(); ok {
			return nil, text, nil
		}
		return nil, item.Label, nil
	}
}

func completionChanges(
	text core.Rope, sel core.Selection, offset *completionEditOffset,
	replace bool, removed string, newText string,
) ([]core.Change, error) {
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		cursor := r.Cursor(text)
		from, to := completionRangeForCursor(text, offset, replace, cursor)
		got, err := text.SliceString(from, to)
		if err != nil {
			return nil, err
		}
		if got != removed {
			from, to = findCompletionRange(text, replace, cursor)
		}
		changes = append(changes, core.TextChange(from, to, newText))
	}
	return changes, nil
}

func completionRangeForCursor(
	text core.Rope, offset *completionEditOffset, replace bool, cursor int,
) (int, int) {
	if from, to, ok := completionRange(text, offset, replace, cursor); ok {
		return from, to
	}
	return findCompletionRange(text, replace, cursor)
}

func completionRange(
	text core.Rope, offset *completionEditOffset, replace bool, cursor int,
) (int, int, bool) {
	if offset == nil {
		from, to := findCompletionRange(text, replace, cursor)
		return from, to, true
	}
	from := cursor + offset.from
	to := cursor + offset.to
	if from < 0 || to > text.LenChars() || from > to {
		return 0, 0, false
	}
	return from, to, true
}

func findCompletionRange(
	text core.Rope, replace bool, cursor int,
) (int, int) {
	before, _ := text.SliceString(0, cursor)
	after, _ := text.SliceString(cursor, text.LenChars())
	from := cursor - countWordSuffix(before)
	to := cursor
	if replace {
		to += countWordPrefix(after)
	}
	return from, to
}

func completionEditRange(
	doc *view.Document, r protocol.Range,
	encoding protocol.PositionEncodingKind,
) (int, int, bool) {
	from, ok := lspPositionToChar(doc, r.Start, encoding)
	if !ok {
		return 0, 0, false
	}
	to, ok := lspPositionToChar(doc, r.End, encoding)
	if !ok {
		return 0, 0, false
	}
	return from, to, true
}

func countWordPrefix(s string) int {
	n := 0
	for _, ch := range s {
		if !core.CharIsWord(ch) {
			return n
		}
		n++
	}
	return n
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
