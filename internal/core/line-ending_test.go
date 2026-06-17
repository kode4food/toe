package core_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestLineEnding(t *testing.T) {
	t.Run("uses native platform ending", func(t *testing.T) {
		e := core.LineEndingLF
		if runtime.GOOS == "windows" {
			e = core.LineEndingCRLF
		}

		assert.Equal(t, e, core.NativeLineEnding())
	})

	t.Run("reports string and character length", func(t *testing.T) {
		assert.Equal(t, "\n", string(core.LineEndingLF))
		assert.Equal(t, "\r\n", string(core.LineEndingCRLF))
		assert.Equal(t, 1, len(core.LineEndingLF))
		assert.Equal(t, 2, len(core.LineEndingCRLF))
	})

	t.Run("parses line ending strings", func(t *testing.T) {
		tests := []struct {
			name string
			in   string
			e    core.LineEnding
			ok   bool
		}{
			{name: "lf", in: "\n", e: core.LineEndingLF, ok: true},
			{name: "crlf", in: "\r\n", e: core.LineEndingCRLF, ok: true},
			{name: "embedded", in: "hello\n", ok: false},
			{name: "carriage return", in: "\r", ok: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e, ok := core.ParseLineEnding(tt.in)

				assert.Equal(t, tt.ok, ok)
				assert.Equal(t, tt.e, e)
			})
		}
	})

	t.Run("recognizes line ending strings", func(t *testing.T) {
		assert.True(t, core.StrIsLineEnding("\n"))
		assert.True(t, core.StrIsLineEnding("\r\n"))
		assert.False(t, core.StrIsLineEnding("\r"))
	})

	t.Run("parses line ending characters", func(t *testing.T) {
		e, ok := core.LineEndingFromChar('\n')
		assert.True(t, ok)
		assert.Equal(t, core.LineEndingLF, e)

		_, ok = core.LineEndingFromChar('\r')
		assert.False(t, ok)
	})

	t.Run("detects first document line ending", func(t *testing.T) {
		tests := []struct {
			name string
			in   string
			e    core.LineEnding
			ok   bool
		}{
			{name: "empty", in: "", ok: false},
			{name: "none", in: "hello", ok: false},
			{name: "lf", in: "\n", e: core.LineEndingLF, ok: true},
			{name: "crlf", in: "\r\n", e: core.LineEndingCRLF, ok: true},
			{
				name: "first ending wins",
				in:   "hello\nsource\r\n",
				e:    core.LineEndingLF,
				ok:   true,
			},
			{
				name: "multiple linefeeds",
				in:   "\n\u000A\n \u000A",
				e:    core.LineEndingLF,
				ok:   true,
			},
			{
				name: "form feed before linefeed",
				in:   "a formfeed\u000C with a\u000C linefeed\u000A",
				e:    core.LineEndingLF,
				ok:   true,
			},
			{
				name: "ignores form feed without unicode lines",
				in:   "a formfeed\u000C with crlf\r\nand lf\n",
				e:    core.LineEndingCRLF,
				ok:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e, ok := core.AutoDetectLineEndingString(tt.in)

				assert.Equal(t, tt.ok, ok)
				assert.Equal(t, tt.e, e)
			})
		}
	})

	t.Run("gets ending from line string", func(t *testing.T) {
		text := "Hello\rworld\nhow\r\nare you?"

		e, ok := core.GetLineEndingOfString(text[:12])
		assert.True(t, ok)
		assert.Equal(t, core.LineEndingLF, e)

		e, ok = core.GetLineEndingOfString(text[:17])
		assert.True(t, ok)
		assert.Equal(t, core.LineEndingCRLF, e)

		_, ok = core.GetLineEndingOfString(text)
		assert.False(t, ok)
	})

	t.Run("removes final line ending", func(t *testing.T) {
		assert.Equal(t, "hello", core.StringWithoutLineEnding("hello\n"))
		assert.Equal(t, "hello", core.StringWithoutLineEnding("hello\r\n"))
		assert.Equal(t, "hello", core.StringWithoutLineEnding("hello"))
	})
}
