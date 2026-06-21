package command

import (
	"fmt"
	"slices"
	"strings"
)

type (
	// Signature describes positional, raw, and flag parsing for a command
	Signature struct {
		Positionals Positionals
		RawAfter    int // 0 = disabled; n = switch to raw after n positionals
		Flags       []Flag
		Completer   Completer
	}

	// ParseError reports a command-line parser validation failure
	ParseError struct {
		Kind   ParseErrorKind
		Token  Token
		Flag   string
		Text   string
		Min    int
		Max    int // 0 = no maximum
		Actual int
	}

	// Flag describes a command flag and optional shorthand
	Flag struct {
		Name        string
		Alias       rune
		Doc         string
		Completions []string
	}

	// Positionals constrains the accepted positional argument count
	Positionals struct {
		Min int
		Max int // 0 = no maximum
	}

	// ParseErrorKind identifies a parser validation failure
	ParseErrorKind int

	// CompletionState describes what kind of argument is being typed
	CompletionState struct {
		Kind CompletionStateKind
		Flag *Flag
	}

	// CompletionStateKind identifies the kind of the last parsed argument
	CompletionStateKind int

	// Args is command-line input interpreted by a command signature
	Args struct {
		signature       Signature
		validate        bool
		onlyPositionals bool
		state           argsCompletionState
		positionals     []string
		flags           map[string]string
	}

	argsCompletionState struct {
		kind argsCompletionKind
		flag *Flag
	}

	argsCompletionKind int
)

const (
	ParseErrorWrongPositionalCount ParseErrorKind = iota
	ParseErrorUnterminatedToken
	ParseErrorDuplicatedFlag
	ParseErrorUnknownFlag
	ParseErrorFlagMissingArgument
	ParseErrorMissingExpansionDelimiter
	ParseErrorUnknownExpansion
)

const (
	argsCompletionPositional argsCompletionKind = iota
	argsCompletionFlag
	argsCompletionFlagArgument
)

const (
	CompletionStateFlag         = CompletionStateKind(argsCompletionFlag)
	CompletionStateFlagArgument = CompletionStateKind(
		argsCompletionFlagArgument,
	)
)

// DefaultSignature returns a signature that accepts any number of positionals
func DefaultSignature() Signature {
	return Signature{Positionals: Positionals{Min: 0}}
}

// ParseArgs parses command input using the supplied signature;
// expand may be nil, in which case raw token content is used as-is
func ParseArgs(
	input string, sig Signature, validate bool, expand TokenExpander,
) (*Args, error) {
	t := NewTokenizer(input, validate)
	args := NewArgs(sig, validate)
	for {
		tok, ok, err := args.readToken(t)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		value := tok.Content
		if expand != nil {
			if value, err = expand(tok); err != nil {
				return nil, err
			}
		}
		if err := args.Push(value); err != nil {
			return nil, err
		}
	}
	if err := args.Finish(); err != nil {
		return nil, err
	}
	return args, nil
}

// NewArgs returns an empty argument accumulator for a signature
func NewArgs(sig Signature, validate bool) *Args {
	return &Args{
		signature:   sig,
		validate:    validate,
		state:       argsCompletionState{kind: argsCompletionPositional},
		positionals: []string{},
		flags:       map[string]string{},
	}
}

// Push adds one already-expanded argument to the accumulator
func (a *Args) Push(arg string) error {
	if !a.onlyPositionals && arg == "--" {
		a.onlyPositionals = true
		a.state = argsCompletionState{kind: argsCompletionFlag}
		return nil
	}
	if flag := a.flagAwaitingArgument(); flag != nil {
		a.flags[flag.Name] = arg
		a.state = argsCompletionState{
			kind: argsCompletionFlagArgument, flag: flag,
		}
		return nil
	}
	if !a.onlyPositionals && strings.HasPrefix(arg, "-") {
		return a.pushFlag(arg)
	}
	a.positionals = append(a.positionals, arg)
	a.state = argsCompletionState{kind: argsCompletionPositional}
	return nil
}

// Finish validates final argument state
func (a *Args) Finish() error {
	if !a.validate {
		return nil
	}
	if flag := a.flagAwaitingArgument(); flag != nil {
		return &ParseError{
			Kind: ParseErrorFlagMissingArgument, Flag: flag.Name,
		}
	}
	return a.signature.checkPositionalCount(len(a.positionals))
}

// Len returns the positional argument count
func (a *Args) Len() int {
	return len(a.positionals)
}

// Empty reports whether there are no positional arguments
func (a *Args) Empty() bool {
	return len(a.positionals) == 0
}

// First returns the first positional argument
func (a *Args) First() (string, bool) {
	return a.Get(0)
}

// Get returns the positional argument at index
func (a *Args) Get(i int) (string, bool) {
	if i < 0 || i >= len(a.positionals) {
		return "", false
	}
	return a.positionals[i], true
}

// Join joins positional arguments with a separator
func (a *Args) Join(sep string) string {
	return strings.Join(a.positionals, sep)
}

// Positionals returns a copy of the positional arguments
func (a *Args) Positionals() []string {
	return slices.Clone(a.positionals)
}

// Flag returns a flag argument value
func (a *Args) Flag(name string) (string, bool) {
	v, ok := a.flags[name]
	return v, ok
}

// HasFlag reports whether a boolean flag was supplied
func (a *Args) HasFlag(name string) bool {
	_, ok := a.flags[name]
	return ok
}

// CompletionState returns what kind of argument the last token was
func (a *Args) CompletionState() CompletionState {
	return CompletionState{
		Kind: CompletionStateKind(a.state.kind),
		Flag: a.state.flag,
	}
}

func (p *ParseError) Error() string {
	switch p.Kind {
	case ParseErrorWrongPositionalCount:
		return p.wrongPositionalCountError()
	case ParseErrorUnterminatedToken:
		return fmt.Sprintf("unterminated token %s", p.Token.Content)
	case ParseErrorDuplicatedFlag:
		return fmt.Sprintf(
			"flag '--%s' specified more than once", p.Flag,
		)
	case ParseErrorUnknownFlag:
		return fmt.Sprintf("unknown flag '%s'", p.Text)
	case ParseErrorFlagMissingArgument:
		return fmt.Sprintf("flag '--%s' missing an argument", p.Flag)
	case ParseErrorMissingExpansionDelimiter:
		if p.Text == "" {
			return "'%' was not properly escaped. Please use '%%'"
		}
		return fmt.Sprintf(
			"missing a string delimiter after '%%%s'", p.Text,
		)
	case ParseErrorUnknownExpansion:
		return fmt.Sprintf("unknown expansion '%s'", p.Text)
	default:
		return ErrCommandLineParse.Error()
	}
}

func (p *ParseError) Is(target error) bool {
	return target == ErrCommandLineParse
}

func (p *ParseError) wrongPositionalCountError() string {
	switch {
	case p.Max > 0 && p.Min == p.Max:
		return fmt.Sprintf(
			"expected exactly %d argument%s, got %d",
			p.Min, plural(p.Min), p.Actual,
		)
	case p.Actual < p.Min:
		return fmt.Sprintf(
			"expected at least %d argument%s, got %d",
			p.Min, plural(p.Min), p.Actual,
		)
	default:
		return fmt.Sprintf(
			"expected at most %d argument%s, got %d",
			p.Max, plural(p.Max), p.Actual,
		)
	}
}

func (s Signature) checkPositionalCount(n int) error {
	lo, hi := s.Positionals.Min, s.Positionals.Max
	if n >= lo && (hi == 0 || n <= hi) {
		return nil
	}
	return &ParseError{
		Kind:   ParseErrorWrongPositionalCount,
		Min:    lo,
		Max:    hi,
		Actual: n,
	}
}

func (a *Args) readToken(
	t *Tokenizer,
) (Token, bool, error) {
	if a.signature.RawAfter > 0 && a.Len() >= a.signature.RawAfter {
		a.onlyPositionals = true
		tok, ok := t.Rest()
		return tok, ok, nil
	}
	return t.Next()
}

func (a *Args) pushFlag(arg string) error {
	flag := a.findFlag(arg)
	if flag == nil {
		if a.validate {
			return &ParseError{Kind: ParseErrorUnknownFlag, Text: arg}
		}
		a.positionals = append(a.positionals, arg)
		a.state = argsCompletionState{kind: argsCompletionFlag}
		return nil
	}
	if a.validate {
		if _, ok := a.flags[flag.Name]; ok {
			return &ParseError{
				Kind: ParseErrorDuplicatedFlag, Flag: flag.Name,
			}
		}
	}
	a.flags[flag.Name] = ""
	a.state = argsCompletionState{kind: argsCompletionFlag, flag: flag}
	return nil
}

func (a *Args) findFlag(arg string) *Flag {
	if name, ok := strings.CutPrefix(arg, "--"); ok {
		for i := range a.signature.Flags {
			if a.signature.Flags[i].Name == name {
				return &a.signature.Flags[i]
			}
		}
		return nil
	}
	alias, ok := strings.CutPrefix(arg, "-")
	if !ok {
		return nil
	}
	for i := range a.signature.Flags {
		f := &a.signature.Flags[i]
		if f.Alias != 0 && alias == string(f.Alias) {
			return f
		}
	}
	return nil
}

func (a *Args) flagAwaitingArgument() *Flag {
	if a.state.kind != argsCompletionFlag || a.state.flag == nil {
		return nil
	}
	if a.state.flag.Completions == nil {
		return nil
	}
	return a.state.flag
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
