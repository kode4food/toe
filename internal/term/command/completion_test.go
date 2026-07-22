package command_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
)

func TestPositionalCompleter(t *testing.T) {
	fn := command.StaticCompleter("alpha", "beta", "gamma")
	c := command.PositionalCompleter(fn)
	assert.Equal(t, 1, len(c.Positionals))
	assert.NotNil(t, c.Positionals[0])
	assert.Nil(t, c.Raw)
}

func TestStaticCompleter(t *testing.T) {
	fn := command.StaticCompleter("foo", "bar", "baz")

	t.Run("matches prefix", func(t *testing.T) {
		got := fn(nil, nil, "ba")
		assert.Equal(t, 2, len(got))
		assert.Equal(t, "bar", got[0].Text)
		assert.Equal(t, "baz", got[1].Text)
	})

	t.Run("empty input matches all", func(t *testing.T) {
		got := fn(nil, nil, "")
		assert.Equal(t, 3, len(got))
	})

	t.Run("no match returns empty", func(t *testing.T) {
		got := fn(nil, nil, "z")
		assert.Equal(t, 0, len(got))
	})
}

func TestCompletePositional(t *testing.T) {
	fn := command.StaticCompleter("one", "two", "three")
	c := command.PositionalCompleter(fn)
	sig := command.Signature{
		Positionals: command.Positionals{Min: 0},
		Completer:   c,
	}

	t.Run("first positional from empty input", func(t *testing.T) {
		got := c.Complete(nil, sig, "")
		assert.Equal(t, 3, len(got))
	})

	t.Run("completes first positional with prefix", func(t *testing.T) {
		got := c.Complete(nil, sig, "t")
		assert.Equal(t, 2, len(got))
		assert.Equal(t, "two", got[0].Text)
		assert.Equal(t, "three", got[1].Text)
	})

	t.Run("completes second positional after space", func(t *testing.T) {
		fn2 := command.StaticCompleter("alpha", "beta")
		c2 := command.PositionalCompleter(fn, fn2)
		sig2 := command.Signature{
			Positionals: command.Positionals{Min: 0},
			Completer:   c2,
		}
		got := c2.Complete(nil, sig2, "one ")
		assert.Equal(t, 2, len(got))
	})

	t.Run("no completer for index returns nil", func(t *testing.T) {
		got := c.Complete(nil, sig, "one two ")
		assert.Nil(t, got)
	})
}

func TestCompleteFlags(t *testing.T) {
	sig := command.Signature{
		Positionals: command.Positionals{Min: 0},
		Flags: []command.Flag{
			{Name: "verbose", Alias: 'v'},
			{Name: "output", Alias: 'o'},
		},
	}
	c := command.Completer{}

	t.Run("completes long flags", func(t *testing.T) {
		got := c.Complete(nil, sig, "--v")
		texts := make([]string, len(got))
		for i, g := range got {
			texts[i] = g.Text
		}
		assert.Contains(t, texts, "--verbose")
	})

	t.Run("completes short flags", func(t *testing.T) {
		got := c.Complete(nil, sig, "-")
		texts := make([]string, len(got))
		for i, g := range got {
			texts[i] = g.Text
		}
		assert.Contains(t, texts, "-v")
		assert.Contains(t, texts, "-o")
	})

	t.Run("completes all flags from empty dash", func(t *testing.T) {
		got := c.Complete(nil, sig, "--")
		texts := make([]string, len(got))
		for i, g := range got {
			texts[i] = g.Text
		}
		assert.Contains(t, texts, "--verbose")
		assert.Contains(t, texts, "--output")
	})
}

func TestCompleteFlagArgument(t *testing.T) {
	sig := command.Signature{
		Positionals: command.Positionals{Min: 0},
		Flags: []command.Flag{
			{
				Name:        "fmt",
				Completions: []string{"json", "yaml", "toml"},
			},
		},
	}
	c := command.Completer{}

	// Flag argument completion triggers when the flag value is being typed
	// (i.e., after --flag has been parsed and a partial value follows)
	t.Run("completes flag argument with prefix j", func(t *testing.T) {
		got := c.Complete(nil, sig, "--fmt j")
		assert.Equal(t, 1, len(got))
		assert.Equal(t, "json", got[0].Text)
	})

	t.Run("completes flag argument with prefix ya", func(t *testing.T) {
		got := c.Complete(nil, sig, "--fmt ya")
		assert.Equal(t, 1, len(got))
		assert.Equal(t, "yaml", got[0].Text)
	})
}

func TestCompleteRaw(t *testing.T) {
	rawFn := command.StaticCompleter("file.go", "file.txt")
	sig := command.Signature{
		Positionals: command.Positionals{Min: 1},
		RawAfter:    1,
	}
	c := command.Completer{Raw: rawFn}

	t.Run("raw completer after rawAfter threshold", func(t *testing.T) {
		got := c.Complete(nil, sig, "cmd file")
		assert.Equal(t, 2, len(got))
	})
}

func TestCompleteOffsets(t *testing.T) {
	fn := command.StaticCompleter("world")
	c := command.PositionalCompleter(fn)
	sig := command.Signature{
		Positionals: command.Positionals{Min: 0},
		Completer:   c,
	}

	t.Run("completion start is offset into input", func(t *testing.T) {
		got := c.Complete(nil, sig, "wor")
		assert.Equal(t, 1, len(got))
		assert.Equal(t, 0, got[0].Start)
	})
}

func TestCompleteNilPositional(t *testing.T) {
	// Nil slot in Positionals list should return nil completions
	c := command.PositionalCompleter(nil)
	sig := command.Signature{
		Positionals: command.Positionals{Min: 0},
		Completer:   c,
	}

	t.Run("nil completer slot returns nil", func(t *testing.T) {
		got := c.Complete(nil, sig, "x")
		assert.Nil(t, got)
	})
}
