package view_test

import (
	"testing"

	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
)

func TestDocumentColors(t *testing.T) {
	e := editorWithText(t, "color: #123456\n")
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)

	colors := []view.DocumentColor{
		{From: 7, To: 14, Red: 0x12, Green: 0x34, Blue: 0x56},
	}
	doc.SetDocumentColors(colors)
	assert.Equal(t, colors, doc.DocumentColors())

	doc.ClearDocumentColors()
	assert.Empty(t, doc.DocumentColors())

	doc.SetDocumentColors(colors)
	doc.SetDocumentColors(nil)
	assert.Empty(t, doc.DocumentColors())
}
