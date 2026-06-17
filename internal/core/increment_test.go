package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

type incrementCase struct {
	input  string
	amount int64
	want   string
}

func TestIncrementInteger(t *testing.T) {
	t.Run("basic decimal increment", func(t *testing.T) {
		tests := []incrementCase{
			{input: "100", amount: 1, want: "101"},
			{input: "100", amount: -1, want: "99"},
			{input: "99", amount: 1, want: "100"},
			{input: "100", amount: 1000, want: "1100"},
			{input: "100", amount: -1000, want: "-900"},
			{input: "-1", amount: 1, want: "0"},
			{input: "-1", amount: 2, want: "1"},
			{input: "1", amount: -1, want: "0"},
			{input: "1", amount: -2, want: "-1"},
		}
		for _, tc := range tests {
			got, ok := core.IncrementInteger(tc.input, tc.amount)
			assert.True(t, ok)
			assert.Equal(t, tc.want, got)
		}
	})

	t.Run("basic hexadecimal increment", func(t *testing.T) {
		tests := []incrementCase{
			{input: "0x0100", amount: 1, want: "0x0101"},
			{input: "0x0100", amount: -1, want: "0x00ff"},
			{input: "0x0001", amount: -1, want: "0x0000"},
			{input: "0x0000", amount: -1, want: "0x0000"},
			{
				input:  "0xABCDEF1234567890",
				amount: 1,
				want:   "0xABCDEF1234567891",
			},
			{
				input:  "0xabcdef1234567890",
				amount: 1,
				want:   "0xabcdef1234567891",
			},
		}
		for _, tc := range tests {
			got, ok := core.IncrementInteger(tc.input, tc.amount)
			assert.True(t, ok)
			assert.Equal(t, tc.want, got)
		}
	})

	t.Run("basic octal increment", func(t *testing.T) {
		tests := []incrementCase{
			{input: "0o0107", amount: 1, want: "0o0110"},
			{input: "0o0110", amount: -1, want: "0o0107"},
			{input: "0o0001", amount: -1, want: "0o0000"},
			{input: "0o7777", amount: 1, want: "0o10000"},
			{input: "0o1000", amount: -1, want: "0o0777"},
			{input: "0o0000", amount: -1, want: "0o0000"},
		}
		for _, tc := range tests {
			got, ok := core.IncrementInteger(tc.input, tc.amount)
			assert.True(t, ok)
			assert.Equal(t, tc.want, got)
		}
	})

	t.Run("basic binary increment", func(t *testing.T) {
		tests := []incrementCase{
			{input: "0b00000100", amount: 1, want: "0b00000101"},
			{input: "0b00000100", amount: -1, want: "0b00000011"},
			{input: "0b00000001", amount: -1, want: "0b00000000"},
			{input: "0b11111111", amount: 1, want: "0b100000000"},
			{input: "0b10000000", amount: -1, want: "0b01111111"},
			{input: "0b0000", amount: -1, want: "0b0000"},
		}
		for _, tc := range tests {
			got, ok := core.IncrementInteger(tc.input, tc.amount)
			assert.True(t, ok)
			assert.Equal(t, tc.want, got)
		}
	})

	t.Run("leading and trailing separators are not valid", func(t *testing.T) {
		_, ok := core.IncrementInteger("9_", 1)
		assert.False(t, ok)
		_, ok = core.IncrementInteger("_9", 1)
		assert.False(t, ok)
		_, ok = core.IncrementInteger("_9_", 1)
		assert.False(t, ok)
	})

	t.Run("empty string returns false", func(t *testing.T) {
		_, ok := core.IncrementInteger("", 1)
		assert.False(t, ok)
	})

	t.Run("non-integer returns false", func(t *testing.T) {
		_, ok := core.IncrementInteger("abc", 1)
		assert.False(t, ok)
	})

	t.Run("with underscore separators", func(t *testing.T) {
		got, ok := core.IncrementInteger("999_999", 1)
		assert.True(t, ok)
		assert.Equal(t, "1_000_000", got)

		got, ok = core.IncrementInteger("1_000_000", -1)
		assert.True(t, ok)
		assert.Equal(t, "999_999", got)
	})
}

func TestIncrementDateTime(t *testing.T) {
	t.Run("increments date by one day", func(t *testing.T) {
		got, ok := core.IncrementDateTime("2021-11-24", 1)
		assert.True(t, ok)
		assert.Equal(t, "2021-11-25", got)
	})

	t.Run("decrements date by one day", func(t *testing.T) {
		got, ok := core.IncrementDateTime("2021-11-24", -1)
		assert.True(t, ok)
		assert.Equal(t, "2021-11-23", got)
	})

	t.Run("increments time by one minute", func(t *testing.T) {
		got, ok := core.IncrementDateTime("23:24", 1)
		assert.True(t, ok)
		assert.Equal(t, "23:25", got)
	})

	t.Run("increments datetime by one minute", func(t *testing.T) {
		got, ok := core.IncrementDateTime("2021-11-24 07:12", 1)
		assert.True(t, ok)
		assert.Equal(t, "2021-11-24 07:13", got)
	})

	t.Run("slash-delimited date", func(t *testing.T) {
		got, ok := core.IncrementDateTime("2021/11/24", 3)
		assert.True(t, ok)
		assert.Equal(t, "2021/11/27", got)
	})

	t.Run("empty string returns false", func(t *testing.T) {
		_, ok := core.IncrementDateTime("", 1)
		assert.False(t, ok)
	})

	t.Run("non-date string returns false", func(t *testing.T) {
		_, ok := core.IncrementDateTime("not a date", 1)
		assert.False(t, ok)
	})
}
