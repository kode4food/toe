package command_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
)

func TestDefaultSignature(t *testing.T) {
	sig := command.DefaultSignature()
	assert.Equal(t, 0, sig.Positionals.Min)
	assert.Equal(t, 0, sig.Positionals.Max)
}

func TestArgsAccessors(t *testing.T) {
	sig := command.DefaultSignature()

	t.Run("empty with no positionals", func(t *testing.T) {
		args := command.NewArgs(sig, false)
		assert.True(t, args.Empty())
		assert.Equal(t, 0, args.Len())
	})

	t.Run("not empty after push", func(t *testing.T) {
		args := command.NewArgs(sig, false)
		err := args.Push("hello")
		assert.NoError(t, err)
		assert.False(t, args.Empty())
		assert.Equal(t, 1, args.Len())
	})

	t.Run("First returns first positional", func(t *testing.T) {
		args := command.NewArgs(sig, false)
		_ = args.Push("alpha")
		_ = args.Push("beta")
		v, ok := args.First()
		assert.True(t, ok)
		assert.Equal(t, "alpha", v)
	})

	t.Run("First returns false when empty", func(t *testing.T) {
		args := command.NewArgs(sig, false)
		_, ok := args.First()
		assert.False(t, ok)
	})

	t.Run("Get returns by index", func(t *testing.T) {
		args := command.NewArgs(sig, false)
		_ = args.Push("a")
		_ = args.Push("b")
		v, ok := args.Get(1)
		assert.True(t, ok)
		assert.Equal(t, "b", v)
	})

	t.Run("Get returns false for out of bounds", func(t *testing.T) {
		args := command.NewArgs(sig, false)
		_, ok := args.Get(0)
		assert.False(t, ok)
		_, ok = args.Get(-1)
		assert.False(t, ok)
	})

	t.Run("Join joins with separator", func(t *testing.T) {
		args := command.NewArgs(sig, false)
		_ = args.Push("x")
		_ = args.Push("y")
		assert.Equal(t, "x/y", args.Join("/"))
	})

	t.Run("Join empty produces empty string", func(t *testing.T) {
		args := command.NewArgs(sig, false)
		assert.Equal(t, "", args.Join(","))
	})
}

func TestArgsCompletionState(t *testing.T) {
	t.Run("positional state after push", func(t *testing.T) {
		sig := command.DefaultSignature()
		args := command.NewArgs(sig, false)
		_ = args.Push("foo")
		cs := args.CompletionState()
		assert.Equal(t, command.CompletionStateKind(0), cs.Kind)
		assert.Nil(t, cs.Flag)
	})

	t.Run("flag state after boolean flag", func(t *testing.T) {
		sig := command.Signature{
			Positionals: command.Positionals{},
			Flags: []command.Flag{
				{Name: "verbose", Alias: 'v'},
			},
		}
		args := command.NewArgs(sig, false)
		_ = args.Push("--verbose")
		cs := args.CompletionState()
		assert.Equal(t, command.CompletionStateFlag, cs.Kind)
	})

	t.Run("completion state after flag with arg", func(t *testing.T) {
		sig := command.Signature{
			Positionals: command.Positionals{},
			Flags: []command.Flag{
				{Name: "fmt", Completions: []string{"json", "yaml"}},
			},
		}
		args := command.NewArgs(sig, false)
		_ = args.Push("--fmt")
		_ = args.Push("json")
		cs := args.CompletionState()
		assert.Equal(t, command.CompletionStateFlagArgument, cs.Kind)
		assert.NotNil(t, cs.Flag)
		assert.Equal(t, "fmt", cs.Flag.Name)
	})
}

func TestParseErrorMessages(t *testing.T) {
	t.Run("wrong count exact", func(t *testing.T) {
		_, err := command.ParseArgs("a b c", command.Signature{
			Positionals: command.Positionals{Min: 2, Max: 2},
		}, true, nil)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "exactly 2 arguments")
	})

	t.Run("wrong count too few", func(t *testing.T) {
		_, err := command.ParseArgs("", command.Signature{
			Positionals: command.Positionals{Min: 3},
		}, true, nil)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "at least 3 arguments")
	})

	t.Run("wrong count too many", func(t *testing.T) {
		_, err := command.ParseArgs("a b c", command.Signature{
			Positionals: command.Positionals{Min: 0, Max: 1},
		}, true, nil)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "at most 1 argument")
	})

	t.Run("singular plural", func(t *testing.T) {
		_, err := command.ParseArgs("a b", command.Signature{
			Positionals: command.Positionals{Min: 1, Max: 1},
		}, true, nil)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "1 argument,")
	})

	t.Run("unknown flag message", func(t *testing.T) {
		_, err := command.ParseArgs("--nope",
			command.DefaultSignature(), true, nil,
		)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "unknown flag")
	})

	t.Run("duplicated flag message", func(t *testing.T) {
		sig := command.Signature{
			Flags: []command.Flag{{Name: "foo"}},
		}
		_, err := command.ParseArgs("--foo --foo", sig, true, nil)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "more than once")
	})

	t.Run("flag missing argument message", func(t *testing.T) {
		sig := command.Signature{
			Flags: []command.Flag{
				{Name: "out", Completions: []string{}},
			},
		}
		_, err := command.ParseArgs("--out", sig, true, nil)
		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
		assert.Contains(t, err.Error(), "missing an argument")
	})
}

func TestCommandArgs(t *testing.T) {
	flags := []command.Flag{
		{Name: "foo", Alias: 'f'},
		{Name: "bar", Alias: 'b', Completions: []string{}},
	}
	sig := command.Signature{
		Positionals: command.Positionals{Min: 1, Max: 2},
		Flags:       flags,
	}

	t.Run("parses flags and positionals", func(t *testing.T) {
		args, err := command.ParseArgs(
			`hello -f -b "xyz 123" world`, sig, true, nil,
		)

		assert.NoError(t, err)
		assert.Equal(t, 2, args.Len())
		assert.True(t, args.HasFlag("foo"))
		v, ok := args.Flag("bar")
		assert.True(t, ok)
		assert.Equal(t, "xyz 123", v)
		assert.Equal(t, []string{"hello", "world"}, args.Positionals())
	})

	t.Run("double dash args become positionals", func(t *testing.T) {
		args, err := command.ParseArgs(
			`hello --bar baz -- --foo`, sig, true, nil,
		)

		assert.NoError(t, err)
		assert.Equal(t, []string{"hello", "--foo"}, args.Positionals())
		assert.False(t, args.HasFlag("foo"))
		v, ok := args.Flag("bar")
		assert.True(t, ok)
		assert.Equal(t, "baz", v)
	})

	t.Run("rejects unknown flags", func(t *testing.T) {
		_, err := command.ParseArgs(`foo --quiz`, sig, true, nil)

		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
	})

	t.Run("rejects duplicated flags", func(t *testing.T) {
		_, err := command.ParseArgs(`--foo bar --foo`, sig, true, nil)

		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
	})

	t.Run("rejects missing flag arguments", func(t *testing.T) {
		_, err := command.ParseArgs(`hello --bar`, sig, true, nil)

		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
	})

	t.Run("rejects wrong positional count", func(t *testing.T) {
		_, err := command.ParseArgs(`--foo`, sig, true, nil)

		assert.True(t, errors.Is(err, command.ErrCommandLineParse))
	})

	t.Run("raw after configured positional count", func(t *testing.T) {
		sig := command.Signature{
			Positionals: command.Positionals{Min: 1, Max: 2},
			RawAfter:    1,
		}

		args, err := command.ParseArgs(
			`gutters ["diff"] ["diff", "diagnostics"]`, sig, true, nil,
		)

		assert.NoError(t, err)
		assert.Equal(t,
			[]string{"gutters", `["diff"] ["diff", "diagnostics"]`},
			args.Positionals())
	})
}
