package view_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type recordingDocumentObserver struct {
	events  []string
	changed view.DocumentChange
}

func TestDocumentObserver(t *testing.T) {
	t.Run("records lifecycle events", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))

		e := view.NewEditor(dir)
		o := &recordingDocumentObserver{}
		e.AddDocumentObserver(o)

		v, err := e.SwitchFile(path)
		assert.NoError(t, err)
		action.InsertChar(e, '/')
		assert.NoError(t, e.Save())
		e.CloseView(v.ID())

		assert.Equal(t,
			[]string{"opened", "changed", "saved", "closed"},
			o.events,
		)
		assert.Equal(t, "package main\n", o.changed.Before.String())
		assert.False(t, o.changed.Changes.Empty())
	})
}

func (o *recordingDocumentObserver) DocumentOpened(*view.Document) {
	o.events = append(o.events, "opened")
}

func (o *recordingDocumentObserver) DocumentChanged(
	_ *view.Document, change view.DocumentChange,
) {
	o.changed = change
	o.events = append(o.events, "changed")
}

func (o *recordingDocumentObserver) DocumentSaved(*view.Document) {
	o.events = append(o.events, "saved")
}

func (o *recordingDocumentObserver) DocumentClosed(*view.Document) {
	o.events = append(o.events, "closed")
}
