package core_test

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestLooksBinary(t *testing.T) {
	utf16leOf := func(s string) []byte {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xFE})
		for _, unit := range utf16.Encode([]rune(s)) {
			buf.WriteByte(byte(unit))
			buf.WriteByte(byte(unit >> 8))
		}
		return buf.Bytes()
	}
	utf16beOf := func(s string) []byte {
		var buf bytes.Buffer
		buf.Write([]byte{0xFE, 0xFF})
		for _, unit := range utf16.Encode([]rune(s)) {
			buf.WriteByte(byte(unit >> 8))
			buf.WriteByte(byte(unit))
		}
		return buf.Bytes()
	}
	utf32leOf := func(s string) []byte {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xFE, 0x00, 0x00})
		for _, r := range s {
			buf.WriteByte(byte(r))
			buf.WriteByte(byte(r >> 8))
			buf.WriteByte(byte(r >> 16))
			buf.WriteByte(byte(r >> 24))
		}
		return buf.Bytes()
	}
	utf32beOf := func(s string) []byte {
		var buf bytes.Buffer
		buf.Write([]byte{0x00, 0x00, 0xFE, 0xFF})
		for _, r := range s {
			buf.WriteByte(byte(r >> 24))
			buf.WriteByte(byte(r >> 16))
			buf.WriteByte(byte(r >> 8))
			buf.WriteByte(byte(r))
		}
		return buf.Bytes()
	}

	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "empty file",
			data: []byte{},
			want: false,
		},
		{
			name: "utf-8 source code",
			data: []byte(
				"package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n",
			),
			want: false,
		},
		{
			name: "utf-8 with BOM",
			data: append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello world\n")...),
			want: false,
		},
		{
			name: "utf-16le text",
			data: utf16leOf("hello world\n"),
			want: false,
		},
		{
			name: "utf-16le surrogate pair",
			data: utf16leOf("hi \U0001F600\n"),
			want: false,
		},
		{
			name: "utf-16be text",
			data: utf16beOf("hello world\n"),
			want: false,
		},
		{
			name: "utf-16le unpaired high surrogate",
			data: append([]byte{0xFF, 0xFE}, 0x00, 0xD8, 'x', 0x00),
			want: true,
		},
		{
			name: "utf-16le lone low surrogate",
			data: append([]byte{0xFF, 0xFE}, 0x00, 0xDC, 'x', 0x00),
			want: true,
		},
		{
			name: "utf-32le text",
			data: utf32leOf("hello world\n"),
			want: false,
		},
		{
			name: "utf-32be text",
			data: utf32beOf("hello world\n"),
			want: false,
		},
		{
			name: "utf-32le out-of-range code point",
			data: append(
				[]byte{0xFF, 0xFE, 0x00, 0x00}, 0x00, 0x00, 0x11, 0x00,
			),
			want: true,
		},
		{
			name: "binary data with NULs",
			data: []byte{0x01, 0x02, 0x00, 0x03, 0x00, 0x04, 0x05},
			want: true,
		},
		{
			name: "control-heavy binary data",
			data: bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x1b}, 64),
			want: true,
		},
		{
			name: "legacy non-utf-8 text (latin-1)",
			data: []byte("caf\xe9 au lait\n"), // "café" in Latin-1
			want: false,
		},
		{
			name: "PNG signature",
			data: []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0},
			want: true,
		},
		{
			name: "gzip signature",
			data: []byte{0x1f, 0x8b, 0x08, 0, 0, 0, 0, 0},
			want: true,
		},
		{
			name: "NUL past sample window",
			data: append(
				bytes.Repeat([]byte("a"), 2*core.BinarySampleSize),
				0x00,
			),
			want: false,
		},
		{
			name: "NUL inside sample window",
			data: append(
				bytes.Repeat([]byte("a"), core.BinarySampleSize/2),
				0x00,
			),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, core.LooksBinary(tt.data))
		})
	}
}

func TestLooksBinaryLargeTextFile(t *testing.T) {
	// a large, ordinary text file should never be flagged
	src := strings.Repeat("the quick brown fox jumps over lazy dog\n", 2000)
	assert.False(t, core.LooksBinary([]byte(src)))
}
