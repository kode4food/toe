package command

import (
	"errors"
	"fmt"
	"strings"
)

type (
	// TokenExpander maps a raw token to its expanded string value;
	// a nil expander uses the token content verbatim
	TokenExpander func(Token) (string, error)

	// SyntaxError is a tokenizer-level parse error
	SyntaxError struct {
		Kind  SyntaxErrorKind
		Token Token
		Text  string
	}

	// Line is an input line split into the command name and the argument text
	// that follows it
	Line struct {
		Name string
		Rest string
	}

	// Token is a token from command-line input
	Token struct {
		Kind         TokenKind
		Expansion    ExpansionKind
		Quote        Quote
		ContentStart int
		Content      string
		Terminated   bool
	}

	// ExpansionKind identifies a percent-token expansion kind
	ExpansionKind int

	// Quote identifies a literal quote delimiter
	Quote int

	// TokenKind identifies how a token should be interpreted
	TokenKind int

	// SyntaxErrorKind identifies a tokenizer-level parse failure
	SyntaxErrorKind int

	// Tokenizer tokenizes command-line input
	Tokenizer struct {
		input    string
		validate bool
		pos      int
	}
)

const (
	ExpansionVariable ExpansionKind = iota
	ExpansionUnicode
	ExpansionShell
	ExpansionRegister
)

const (
	QuoteSingle Quote = iota
	QuoteBacktick
)

const (
	TokenUnquoted TokenKind = iota
	TokenQuoted
	TokenExpand
	TokenExpansion
	TokenExpansionKind
)

const (
	SyntaxErrorUnterminatedToken SyntaxErrorKind = iota
	SyntaxErrorMissingExpansionDelimiter
	SyntaxErrorUnknownExpansion
)

var (
	// ErrCommandLineParse is the sentinel error for command-line parse failures
	ErrCommandLineParse = errors.New("command line parse error")
)

// NewTokenizer returns a tokenizer for command-line input
func NewTokenizer(input string, validate bool) *Tokenizer {
	return &Tokenizer{input: input, validate: validate}
}

// Error describes where the command line failed to tokenize
func (s *SyntaxError) Error() string {
	switch s.Kind {
	case SyntaxErrorUnterminatedToken:
		return fmt.Sprintf("unterminated token %s", s.Token.Content)
	case SyntaxErrorMissingExpansionDelimiter:
		if s.Text == "" {
			return "'%' was not properly escaped. Please use '%%'"
		}
		return fmt.Sprintf(
			"missing a string delimiter after '%%%s'", s.Text,
		)
	case SyntaxErrorUnknownExpansion:
		return fmt.Sprintf("unknown expansion '%s'", s.Text)
	default:
		return ErrCommandLineParse.Error()
	}
}

// Is matches the shared command-line parse sentinel
func (s *SyntaxError) Is(target error) bool {
	return target == ErrCommandLineParse
}

// Pos is the offset the tokenizer has consumed to
func (t *Tokenizer) Pos() int {
	return t.pos
}

// Rest consumes everything left as one unterminated expansion token, for
// commands taking a raw trailing argument
func (t *Tokenizer) Rest() (Token, bool) {
	t.skipBlanks()
	if t.pos == len(t.input) {
		return Token{}, false
	}
	start := t.pos
	t.pos = len(t.input)
	return Token{
		Kind:         TokenExpand,
		ContentStart: start,
		Content:      t.input[start:],
	}, true
}

// Next consumes the next token, reporting false at end of input
func (t *Tokenizer) Next() (Token, bool, error) {
	t.skipBlanks()
	if t.pos == len(t.input) {
		return Token{}, false, nil
	}

	b := t.input[t.pos]
	switch b {
	case '"', '\'', '`':
		tok := t.parseQuoteToken(b)
		if t.validate && !tok.Terminated {
			return Token{}, false, &SyntaxError{
				Kind:  SyntaxErrorUnterminatedToken,
				Token: tok,
			}
		}
		return tok, true, nil
	case '%':
		tok, err := t.parsePercentToken()
		return tok, err == nil, err
	default:
		start := t.pos
		if backslashEscapes && b == '\\' && t.peekEscapedToken() {
			t.pos++
		}
		return Token{
			Kind:         TokenUnquoted,
			ContentStart: start,
			Content:      t.parseUnquoted(),
		}, true, nil
	}
}

func (t *Tokenizer) skipBlanks() {
	for t.pos < len(t.input) {
		if t.input[t.pos] != ' ' && t.input[t.pos] != '\t' {
			return
		}
		t.pos++
	}
}

func (t *Tokenizer) parseUnquoted() string {
	var b strings.Builder
	start := t.pos
	escaped := false
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if ch == ' ' || ch == '\t' {
			if backslashEscapes && t.prevByte() == '\\' {
				b.WriteString(t.input[start : t.pos-1])
				b.WriteByte(ch)
				t.pos++
				start = t.pos
				escaped = true
				continue
			} else if !escaped {
				return t.input[start:t.pos]
			}
			break
		}
		t.pos++
	}

	end := t.pos
	if backslashEscapes && t.prevByte() == '\\' {
		end--
	}
	if !escaped {
		return t.input[start:end]
	}
	b.WriteString(t.input[start:end])
	return b.String()
}

func (t *Tokenizer) parseQuoteToken(quote byte) Token {
	start := t.pos + 1
	content, terminated := t.parseQuoted(quote)
	tok := Token{
		ContentStart: start,
		Content:      content,
		Terminated:   terminated,
	}
	switch quote {
	case '"':
		tok.Kind = TokenExpand
	case '\'':
		tok.Kind = TokenQuoted
		tok.Quote = QuoteSingle
	default:
		tok.Kind = TokenQuoted
		tok.Quote = QuoteBacktick
	}
	return tok
}

func (t *Tokenizer) parseQuoted(quote byte) (string, bool) {
	t.pos++
	var b strings.Builder
	start := t.pos
	escaped := false
	for t.pos < len(t.input) {
		idx := strings.IndexByte(t.input[t.pos:], quote)
		if idx < 0 {
			break
		}
		idx += t.pos
		if idx+1 < len(t.input) && t.input[idx+1] == quote {
			b.WriteString(t.input[start : idx+1])
			t.pos = idx + 2
			start = t.pos
			escaped = true
			continue
		}
		if !escaped {
			content := t.input[start:idx]
			t.pos = idx + 1
			return content, true
		}
		b.WriteString(t.input[start:idx])
		t.pos = idx + 1
		return b.String(), true
	}
	if !escaped {
		content := t.input[start:]
		t.pos = len(t.input)
		return content, false
	}
	b.WriteString(t.input[start:])
	t.pos = len(t.input)
	return b.String(), false
}

func (t *Tokenizer) byte() byte {
	if t.pos >= len(t.input) {
		return 0
	}
	return t.input[t.pos]
}

func (t *Tokenizer) prevByte() byte {
	if t.pos == 0 {
		return 0
	}
	return t.input[t.pos-1]
}

func (t *Tokenizer) peekEscapedToken() bool {
	if t.pos+1 >= len(t.input) {
		return false
	}
	switch t.input[t.pos+1] {
	case '"', '\'', '`', '%':
		return true
	default:
		return false
	}
}

// SplitCommandLine separates the command name from its argument text. The
// bool reports whether the name is still being typed
func SplitCommandLine(input string) (Line, bool) {
	i := strings.IndexAny(input, " \t")
	if i < 0 {
		return Line{Name: input}, true
	}
	name := input[:i]
	rest := input[i+1:]
	complete := name == "" ||
		(strings.TrimSpace(rest) == "" && !strings.HasSuffix(input, " ") &&
			!strings.HasSuffix(input, "\t"))
	return Line{Name: name, Rest: rest}, complete
}
