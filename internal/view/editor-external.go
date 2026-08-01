package view

import (
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/loader"
)

// ProcessExternalFileChange updates any open document whose backing file
// changed outside the editor
func (e *Editor) ProcessExternalFileChange(path string) bool {
	target := loader.CanonicalPath(path)
	handled := false
	for _, doc := range e.documents.byID {
		if loader.CanonicalPath(doc.Path()) != target {
			continue
		}
		if e.processExternalChange(doc) {
			handled = true
		}
	}
	return handled
}

func (e *Editor) processExternalChange(doc *Document) bool {
	snap, changed := doc.diskChanged()
	if !changed {
		return false
	}
	if !snap.exists {
		doc.file.snapshot = snap
		doc.file.external = ExternalStateDeleted
		e.SetStatusMsg(i18n.Text(i18n.StatusFileDeleted, i18n.Vars{
			"file": doc.RelativeName(e.workspace.cwd),
		}))
		return true
	}
	if doc.Modified() {
		doc.file.snapshot = snap
		doc.file.external = ExternalStateChanged
		e.SetStatusMsg(i18n.Text(i18n.StatusFileChanged, i18n.Vars{
			"file": doc.RelativeName(e.workspace.cwd),
		}))
		return true
	}

	before := doc.Text()
	rev := doc.Revision()
	if err := doc.reloadPreservingSelections(); err != nil {
		e.SetStatusMsg(i18n.Text(i18n.StatusReloadFailed, i18n.Vars{
			"error": err,
		}))
		return true
	}
	if doc.Revision() != rev {
		e.documentChanged(doc, wholeDocumentChange(before, doc.Text().String()))
	}
	e.SetStatusMsg(i18n.Text(i18n.StatusFileReloaded, i18n.Vars{
		"file": doc.RelativeName(e.workspace.cwd),
	}))
	return true
}
