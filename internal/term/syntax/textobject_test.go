package syntax_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/syntax"
)

const goSrc = "package main\n\nfunc foo(x int) int {\n\treturn x + 1\n}\n" +
	"\ntype Bar struct {\n\tName string\n}\n" +
	"\nfunc (b Bar) Greet() {\n\tfmt.Println(b.Name)\n}\n"

func TestFindTextObjectFunction(t *testing.T) {
	cursor := strings.Index(goSrc, "return")
	runes := []rune(goSrc)

	t.Run("around selects full declaration", func(t *testing.T) {
		r, ok := syntax.FindTextObject(goSrc, "go", cursor, 'f', false)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "func foo")
		assert.Contains(t, got, "return x + 1")
	})

	t.Run("inside selects body without braces", func(t *testing.T) {
		r, ok := syntax.FindTextObject(goSrc, "go", cursor, 'f', true)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "return x + 1")
		assert.NotContains(t, got, "func foo")
		assert.NotContains(t, got, "{")
	})
}

func TestFindTextObjectMethod(t *testing.T) {
	cursor := strings.Index(goSrc, "Println")
	runes := []rune(goSrc)

	t.Run("around selects method declaration", func(t *testing.T) {
		r, ok := syntax.FindTextObject(goSrc, "go", cursor, 'f', false)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "func (b Bar) Greet")
		assert.Contains(t, got, "Println")
	})

	t.Run("inside selects method body", func(t *testing.T) {
		r, ok := syntax.FindTextObject(goSrc, "go", cursor, 'f', true)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "Println")
		assert.NotContains(t, got, "func")
	})
}

func TestFindTextObjectType(t *testing.T) {
	cursor := strings.Index(goSrc, "Name string")
	runes := []rune(goSrc)

	t.Run("around selects type declaration", func(t *testing.T) {
		r, ok := syntax.FindTextObject(goSrc, "go", cursor, 't', false)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "type Bar struct")
		assert.Contains(t, got, "Name string")
	})

	t.Run("inside selects struct body", func(t *testing.T) {
		r, ok := syntax.FindTextObject(goSrc, "go", cursor, 't', true)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "Name string")
		assert.NotContains(t, got, "type Bar")
	})
}

func TestFindTextObjectParameter(t *testing.T) {
	cursor := strings.Index(goSrc, "x int")
	runes := []rune(goSrc)

	t.Run("around selects parameter list with parens", func(t *testing.T) {
		r, ok := syntax.FindTextObject(goSrc, "go", cursor, 'a', false)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "(")
		assert.Contains(t, got, "x int")
		assert.Contains(t, got, ")")
	})

	t.Run("inside selects parameters without parens", func(t *testing.T) {
		r, ok := syntax.FindTextObject(goSrc, "go", cursor, 'a', true)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "x int")
		assert.NotContains(t, got, "(")
	})
}

func TestFindTextObjectCall(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tfmt.Println(alpha, beta)\n}\n"
	cursor := strings.Index(src, "alpha")
	runes := []rune(src)

	t.Run("around selects full call", func(t *testing.T) {
		r, ok := syntax.FindTextObject(src, "go", cursor, 'c', false)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "fmt.Println")
		assert.Contains(t, got, "alpha")
	})

	t.Run("inside selects arguments without parens", func(t *testing.T) {
		r, ok := syntax.FindTextObject(src, "go", cursor, 'c', true)
		assert.True(t, ok)
		got := string(runes[r.From:r.To])
		assert.Contains(t, got, "alpha")
		assert.NotContains(t, got, "fmt.Println")
	})
}

func TestFindTextObjectUnknownLang(t *testing.T) {
	_, ok := syntax.FindTextObject("func foo() {}", "unknown", 5, 'f', true)
	assert.False(t, ok)
}

func TestFindTextObjectUnknownChar(t *testing.T) {
	_, ok := syntax.FindTextObject(goSrc, "go", 20, 'z', true)
	assert.False(t, ok)
}

func TestFindTextObjectNoMatch(t *testing.T) {
	src := "package main\n\nvar x = 1\n"
	cursor := strings.Index(src, "x")
	_, ok := syntax.FindTextObject(src, "go", cursor, 'f', true)
	assert.False(t, ok)
}

func TestIsTextObjectChar(t *testing.T) {
	assert.True(t, syntax.IsTextObjectChar('f'))
	assert.True(t, syntax.IsTextObjectChar('t'))
	assert.True(t, syntax.IsTextObjectChar('a'))
	assert.True(t, syntax.IsTextObjectChar('c'))
	assert.True(t, syntax.IsTextObjectChar('e'))
	assert.False(t, syntax.IsTextObjectChar('w'))
	assert.False(t, syntax.IsTextObjectChar('('))
	assert.False(t, syntax.IsTextObjectChar('z'))
}
