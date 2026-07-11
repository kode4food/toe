package lsp

import (
	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
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

	completionEditOffset struct {
		from int
		to   int
	}
)

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
