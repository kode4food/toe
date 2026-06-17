package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestMode(t *testing.T) {
	assert.Equal(t, "NOR", view.ModeNormal.String())
	assert.Equal(t, "INS", view.ModeInsert.String())
	assert.Equal(t, "SEL", view.ModeSelect.String())
	assert.Equal(t, "NOR", view.Mode(99).String())
}

func TestDocumentDisplayName(t *testing.T) {
	assert.Equal(t, "file.txt", view.DocumentDisplayName("/a/b/file.txt"))
	assert.Equal(t, view.ScratchBufferName, view.DocumentDisplayName(""))
}

func TestJumpList(t *testing.T) {
	t.Run("empty list backward returns false", func(t *testing.T) {
		var j view.JumpList
		_, _, ok := j.Backward()
		assert.False(t, ok)
	})

	t.Run("empty list forward returns false", func(t *testing.T) {
		var j view.JumpList
		_, _, ok := j.Forward()
		assert.False(t, ok)
	})

	t.Run("push then backward", func(t *testing.T) {
		var j view.JumpList
		j.Push(view.DocumentId(1), 0)
		j.Push(view.DocumentId(1), 5)
		docID, anchor, ok := j.Backward()
		assert.True(t, ok)
		assert.Equal(t, view.DocumentId(1), docID)
		assert.Equal(t, 0, anchor)
	})

	t.Run("forward after backward", func(t *testing.T) {
		var j view.JumpList
		j.Push(view.DocumentId(1), 0)
		j.Push(view.DocumentId(1), 5)
		j.Backward()
		docID, anchor, ok := j.Forward()
		assert.True(t, ok)
		assert.Equal(t, view.DocumentId(1), docID)
		assert.Equal(t, 5, anchor)
	})

	t.Run("push discards forward history", func(t *testing.T) {
		var j view.JumpList
		j.Push(view.DocumentId(1), 0)
		j.Push(view.DocumentId(1), 5)
		j.Backward()
		j.Push(view.DocumentId(1), 10)
		_, _, ok := j.Forward()
		assert.False(t, ok)
	})
}

func TestDocumentOpenError(t *testing.T) {
	err := &view.DocumentOpenError{Path: "foo.txt", Err: assert.AnError}
	assert.Equal(t, "foo.txt", err.Path)
	assert.Equal(t, assert.AnError, err.Unwrap())
	assert.Contains(t, err.Error(), "foo.txt")
}

func TestJumpListEntries(t *testing.T) {
	t.Run("entries returns all pushed items", func(t *testing.T) {
		var j view.JumpList
		j.Push(view.DocumentId(1), 0)
		j.Push(view.DocumentId(2), 10)
		entries := j.Entries()
		assert.Equal(t, 2, len(entries))
		assert.Equal(t, view.DocumentId(1), entries[0].DocID)
		assert.Equal(t, 0, entries[0].Anchor)
		assert.Equal(t, view.DocumentId(2), entries[1].DocID)
		assert.Equal(t, 10, entries[1].Anchor)
	})

	t.Run("empty list returns empty slice", func(t *testing.T) {
		var j view.JumpList
		assert.Empty(t, j.Entries())
	})
}

func TestDocumentRelativeNameEdgeCases(t *testing.T) {
	t.Run("path outside basedir returns absolute", func(t *testing.T) {
		name := view.DocumentRelativeName("/other/file.txt", "/base")
		assert.Equal(t, "/other/file.txt", name)
	})
}
