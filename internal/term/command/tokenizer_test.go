package command_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
)

func TestCommandLine(t *testing.T) {
	t.Run("tokenizes unquoted arguments", func(t *testing.T) {
		args, err := commandTokens("hello\t \tworld", true)

		assert.NoError(t, err)
		assert.Equal(t, []string{"hello", "world"}, args)
	})

	t.Run("tokenizes quoted arguments", func(t *testing.T) {
		args, err := commandTokens(`echo "hello "" world"`, true)

		assert.NoError(t, err)
		assert.Equal(t, []string{"echo", `hello " world`}, args)
	})

	t.Run("parses doubled quote escape", func(t *testing.T) {
		args, err := commandTokens("'a '' b'", true)

		assert.NoError(t, err)
		assert.Equal(t, []string{"a ' b"}, args)
	})

	t.Run("parses escaped blank in unquoted token", func(t *testing.T) {
		args, err := commandTokens(`open a\ b.txt`, true)

		assert.NoError(t, err)
		assert.Equal(t, []string{"open", "a b.txt"}, args)
	})

	t.Run("parses percent expansion delimiters", func(t *testing.T) {
		args, err := commandTokens(`echo %{hello {x} world}`, true)

		assert.NoError(t, err)
		assert.Equal(t, []string{"echo", "hello {x} world"}, args)
	})

	t.Run("incomplete tokens without validation", func(t *testing.T) {
		args, err := commandTokens(`echo %sh{echo "%{c`, false)

		assert.NoError(t, err)
		assert.Equal(t, []string{"echo", `echo "%{c`}, args)
	})

	t.Run("rejects unterminated quote", func(t *testing.T) {
		_, err := commandTokens("'open", true)

		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
	})
}

func TestSplitCommandLine(t *testing.T) {
	t.Run("splits command and args", func(t *testing.T) {
		name, rest, complete := command.SplitCommandLine("open file")

		assert.Equal(t, "open", name)
		assert.Equal(t, "file", rest)
		assert.False(t, complete)
	})

	t.Run("detects incomplete command name", func(t *testing.T) {
		name, rest, complete := command.SplitCommandLine("op")

		assert.Equal(t, "op", name)
		assert.Equal(t, "", rest)
		assert.True(t, complete)
	})

	t.Run("detects completed command name", func(t *testing.T) {
		name, rest, complete := command.SplitCommandLine("open ")

		assert.Equal(t, "open", name)
		assert.Equal(t, "", rest)
		assert.False(t, complete)
	})
}

func TestSyntaxErrorMessages(t *testing.T) {
	t.Run("unterminated token", func(t *testing.T) {
		_, err := commandTokens("'hello", true)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "unterminated token")
	})

	t.Run("missing delimiter after bare percent", func(t *testing.T) {
		_, err := commandTokens("%x", true)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "missing a string delimiter")
	})

	t.Run("missing delimiter after named expansion", func(t *testing.T) {
		_, err := commandTokens("%sh", true)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t,
			err.Error(), "missing a string delimiter after '%sh'",
		)
	})

	t.Run("bare percent not escaped", func(t *testing.T) {
		_, err := commandTokens("%", true)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "Please use '%%'")
	})

	t.Run("unknown expansion", func(t *testing.T) {
		_, err := commandTokens("%zzz{val}", true)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "unknown expansion")
	})
}

func TestTokenizerPos(t *testing.T) {
	t.Run("pos advances with tokens", func(t *testing.T) {
		tok := command.NewTokenizer("ab cd", true)
		_, _, _ = tok.Next()
		assert.Equal(t, 2, tok.Pos())
	})

	t.Run("pos is zero initially", func(t *testing.T) {
		tok := command.NewTokenizer("", true)
		assert.Equal(t, 0, tok.Pos())
	})
}

func TestPeekEscapedToken(t *testing.T) {
	t.Run("backslash before quote parsed as escape", func(t *testing.T) {
		// On non-windows, backslash before ' is an escape
		args, err := commandTokens(`\'hello`, true)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(args))
	})

	t.Run("backslash before double-quote escape", func(t *testing.T) {
		args, err := commandTokens(`\"hello`, true)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(args))
	})

	t.Run("backslash before backtick escape", func(t *testing.T) {
		args, err := commandTokens("`hello`", false)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(args))
		assert.Equal(t, "hello", args[0])
	})

	t.Run("backslash before percent escape", func(t *testing.T) {
		args, err := commandTokens(`\%`, true)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(args))
	})
}

func TestTokenizerIncomplete(t *testing.T) {
	t.Run("incomplete expansion kind no-validate", func(t *testing.T) {
		args, err := commandTokens("%zzz", false)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(args))
	})

	t.Run("pipe delimiter expansion", func(t *testing.T) {
		args, err := commandTokens("%|hello|", true)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(args))
		assert.Equal(t, "hello", args[0])
	})
}
