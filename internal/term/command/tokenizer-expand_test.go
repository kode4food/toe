package command_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpansionDelimiters(t *testing.T) {
	cases := []struct {
		input    string
		contains string
	}{
		{`%{hello}`, "hello"},
		{`%(hello)`, "hello"},
		{`%[hello]`, "hello"},
		{`%<hello>`, "hello"},
		{`%'hello'`, "hello"},
		{`%"hello"`, "hello"},
		{`%|hello|`, "hello"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			args, err := commandTokens(tc.input, true)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(args))
			assert.Equal(t, tc.contains, args[0])
		})
	}
}

func TestCommandExpansionKind(t *testing.T) {
	cases := []struct {
		input string
	}{
		{`%{var}`},
		{`%u{0041}`},
		{`%sh{echo hello}`},
		{`%reg{a}`},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			args, err := commandTokens(tc.input, true)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(args))
		})
	}
}
