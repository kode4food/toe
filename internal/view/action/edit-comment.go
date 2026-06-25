package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

// ToggleComments toggles the most appropriate comment style for the language:
// block comments if already block-commented, line comments otherwise
func ToggleComments(e *view.Editor) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	lang := language.LoadLanguage(doc.Lang())
	lineToken, hasLine := firstCommentToken(lang)
	blockTokens := lang.BlockCommentTokens
	if hasLine && len(blockTokens) == 0 {
		toggleLineCommentsWithToken(e, lineToken)
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	split, err := core.SplitLinesOfSelection(text, sel)
	if err != nil {
		return
	}
	tokens := blockTokens
	lineCommented, _, err := core.FindBlockComments(tokens, text, split)
	if err != nil {
		return
	}
	if lineCommented {
		toggleBlockCommentsSelection(e, split, tokens)
		return
	}
	blockCommented, _, err := core.FindBlockComments(tokens, text, sel)
	if err != nil {
		return
	}
	if blockCommented {
		toggleBlockCommentsSelection(e, sel, tokens)
		return
	}
	if !hasLine && len(blockTokens) > 0 {
		toggleBlockCommentsSelection(e, split, tokens)
		return
	}
	toggleLineCommentsWithToken(e, lineToken)
}

// ToggleLineComments toggles line comments on the selected lines
func ToggleLineComments(e *view.Editor) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	lang := language.LoadLanguage(doc.Lang())
	lineToken, hasLine := firstCommentToken(lang)
	if hasLine {
		toggleLineCommentsWithToken(e, lineToken)
		return
	}
	if len(lang.BlockCommentTokens) > 0 {
		v, ok := e.FocusedView()
		if !ok {
			return
		}
		sel, err := core.SplitLinesOfSelection(
			doc.Text(), doc.SelectionFor(v.ID()),
		)
		if err != nil {
			return
		}
		toggleBlockCommentsSelection(e, sel, lang.BlockCommentTokens)
		return
	}
	toggleLineCommentsWithToken(e, "")
}

// ToggleBlockComments toggles block comments on each selection range
func ToggleBlockComments(e *view.Editor) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	lang := language.LoadLanguage(doc.Lang())
	switch {
	case len(lang.BlockCommentTokens) > 0:
		toggleBlockCommentsWithTokens(e, lang.BlockCommentTokens)
	case len(lang.CommentTokens) > 0:
		toggleLineCommentsWithToken(e, lang.CommentTokens[0])
	default:
		toggleBlockCommentsWithTokens(e, nil)
	}
}

func firstCommentToken(lang *language.Language) (string, bool) {
	if len(lang.CommentTokens) == 0 {
		return "", false
	}
	return lang.CommentTokens[0], true
}

func toggleLineCommentsWithToken(e *view.Editor, token string) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	tx, err := core.ToggleLineComments(
		doc.Text(), doc.SelectionFor(v.ID()), token,
	)
	if err != nil {
		return
	}
	if err := e.Apply(tx); err == nil {
		ExitSelectMode(e)
	}
}

func toggleBlockCommentsWithTokens(
	e *view.Editor, tokens []core.BlockCommentToken,
) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	toggleBlockCommentsSelection(e, doc.SelectionFor(v.ID()), tokens)
}

func toggleBlockCommentsSelection(
	e *view.Editor, sel core.Selection, tokens []core.BlockCommentToken,
) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	tx, err := core.ToggleBlockComments(
		doc.Text(), sel, tokens,
	)
	if err != nil {
		return
	}
	if err := e.Apply(tx); err == nil {
		ExitSelectMode(e)
	}
}
