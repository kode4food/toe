package view

import "path/filepath"

// ProcessExternalFileChange updates any open document whose backing file
// changed outside the editor
func (e *Editor) ProcessExternalFileChange(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	handled := false
	for _, doc := range e.docs {
		if doc.Path() != abs {
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
		doc.disk = snap
		doc.external = ExternalStateDeleted
		e.SetStatusMsg("'" + doc.RelativeName(e.cwd) + "' deleted on disk")
		return true
	}
	if doc.Modified() {
		doc.disk = snap
		doc.external = ExternalStateChanged
		e.SetStatusMsg(
			"'" + doc.RelativeName(e.cwd) +
				"' changed on disk; use :reload or :write",
		)
		return true
	}

	before := doc.Text()
	rev := doc.Revision()
	if err := doc.reloadPreservingSelections(); err != nil {
		e.SetStatusMsg("reload failed: " + err.Error())
		return true
	}
	if doc.Revision() != rev {
		e.documentChanged(doc, wholeDocumentChange(before, doc.Text().String()))
	}
	e.SetStatusMsg("'" + doc.RelativeName(e.cwd) + "' reloaded")
	return true
}
