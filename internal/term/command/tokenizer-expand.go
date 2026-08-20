package command

import "strings"

type delimiterPair struct {
	open  byte
	close byte
}

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
	for t.pos < len(t.input) && isLowerASCII(t.input[t.pos]) {
		t.pos++
	}
	kindText := t.input[kindStart:t.pos]
	delims, ok := expansionDelimiters(t.byte())
	if !ok {
		tok := Token{
			Kind:         TokenExpansionKind,
			ContentStart: kindStart,
			Content:      kindText,
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
	kind, ok := expansionKind(kindText)
	if !ok && t.validate {
		return Token{}, &SyntaxError{
			Kind: SyntaxErrorUnknownExpansion,
			Text: kindText,
		}
	}
	content, terminated := t.parseDelimited(delims)
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

func (t *Tokenizer) parseDelimited(delims delimiterPair) (string, bool) {
	open := delims.open
	if open == delims.close {
		return t.parseQuoted(open)
	}
	t.pos++
	start := t.pos
	level := 1
	for t.pos < len(t.input) {
		idx := strings.IndexAny(
			t.input[t.pos:], string([]byte{open, delims.close}),
		)
		if idx < 0 {
			break
		}
		idx += t.pos
		t.pos = idx + 1
		if t.input[idx] == open {
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

func expansionKind(name string) (ExpansionKind, bool) {
	k, ok := expansionKindNames[name]
	return k, ok
}

func isLowerASCII(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

func expansionDelimiters(ch byte) (delimiterPair, bool) {
	if pair, ok := expansionDelimPairs[ch]; ok {
		return delimiterPair{open: pair[0], close: pair[1]}, true
	}
	return delimiterPair{}, false
}
