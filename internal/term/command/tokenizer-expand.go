package command

import "strings"

var (
	expansionKindNames = map[string]ExpansionKind{
		"":    ExpansionVariable,
		"u":   ExpansionUnicode,
		"sh":  ExpansionShell,
		"reg": ExpansionRegister,
	}

	expansionDelimPairs = map[byte][2]byte{
		'(':  {'(', ')'},
		'[':  {'[', ']'},
		'{':  {'{', '}'},
		'<':  {'<', '>'},
		'\'': {'\'', '\''},
		'"':  {'"', '"'},
		'|':  {'|', '|'},
	}
)

func (t *Tokenizer) parsePercentToken() (Token, error) {
	t.pos++
	kindStart := t.pos
	for t.pos < len(t.input) && lowerASCII(t.input[t.pos]) {
		t.pos++
	}
	kindText := t.input[kindStart:t.pos]
	start, end, ok := expansionDelimiters(t.byte())
	if !ok {
		tok := Token{
			Kind:         TokenExpansionKind,
			ContentStart: kindStart,
			Content:      kindText,
			Terminated:   false,
		}
		if !t.validate {
			return tok, nil
		}
		return Token{}, &SyntaxError{
			Kind: SyntaxErrorMissingExpansionDelimiter,
			Text: kindText,
		}
	}

	contentStart := t.pos + 1
	kind, ok := commandExpansionKind(kindText)
	if !ok && t.validate {
		return Token{}, &SyntaxError{
			Kind: SyntaxErrorUnknownExpansion,
			Text: kindText,
		}
	}
	content, terminated := t.parseDelimited(start, end)
	tok := Token{
		Kind:         TokenExpansion,
		Expansion:    kind,
		ContentStart: contentStart,
		Content:      content,
		Terminated:   terminated,
	}
	if !ok {
		tok.Kind = TokenExpand
	}
	if t.validate && !terminated {
		return Token{}, &SyntaxError{
			Kind:  SyntaxErrorUnterminatedToken,
			Token: tok,
		}
	}
	return tok, nil
}

func (t *Tokenizer) parseDelimited(openDelim, closeDelim byte) (string, bool) {
	if openDelim == closeDelim {
		return t.parseQuoted(openDelim)
	}
	t.pos++
	start := t.pos
	level := 1
	for t.pos < len(t.input) {
		idx := strings.IndexAny(
			t.input[t.pos:], string([]byte{openDelim, closeDelim}),
		)
		if idx < 0 {
			break
		}
		idx += t.pos
		t.pos = idx + 1
		if t.input[idx] == openDelim {
			level++
			continue
		}
		level--
		if level == 0 {
			return t.input[start:idx], true
		}
	}
	t.pos = len(t.input)
	return t.input[start:], false
}

func commandExpansionKind(name string) (ExpansionKind, bool) {
	k, ok := expansionKindNames[name]
	return k, ok
}

func lowerASCII(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

func expansionDelimiters(ch byte) (byte, byte, bool) {
	if pair, ok := expansionDelimPairs[ch]; ok {
		return pair[0], pair[1], true
	}
	return 0, 0, false
}
