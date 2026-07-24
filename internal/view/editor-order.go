package view

// LastModifiedDocIDs returns the two most recently modified-and-left documents,
// with the most recent first. Invalid entries have value InvalidDocumentId
func (e *Editor) LastModifiedDocIDs() [2]DocumentId {
	return e.documents.lastModifiedIDs
}

// PopPrevDocID returns and removes the most recently accessed document for the
// focused view
func (e *Editor) PopPrevDocID() (DocumentId, bool) {
	v, ok := e.FocusedView()
	if !ok {
		return InvalidDocumentId, false
	}
	for len(v.docHistory) > 0 {
		last := len(v.docHistory) - 1
		did := v.docHistory[last]
		v.docHistory = v.docHistory[:last]
		if did == v.DocID() {
			continue
		}
		if _, ok := e.documents.byID[did]; ok {
			return did, true
		}
	}
	return InvalidDocumentId, false
}

// recordPrevDoc adds the current document to the focused view's access history
// before replacing it
func (e *Editor) recordPrevDoc() {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	v.addDocHistory(v.DocID())
	e.recordLeavingDocFor(v)
}

func (e *Editor) recordLeavingDoc() {
	if v, ok := e.FocusedView(); ok {
		e.recordLeavingDocFor(v)
	}
}

func (e *Editor) recordLeavingDocFor(v *View) {
	doc, ok := e.documents.byID[v.DocID()]
	if !ok {
		return
	}
	doc.rememberSelection(v.ID())
	if doc.edits.changedSinceAccess {
		did := doc.ID()
		if e.documents.lastModifiedIDs[0] != did {
			e.documents.lastModifiedIDs[1] = e.documents.lastModifiedIDs[0]
			e.documents.lastModifiedIDs[0] = did
		}
	}
}

func (e *Editor) markDocAccessed() {
	if v, ok := e.FocusedView(); ok {
		if doc, ok := e.documents.byID[v.DocID()]; ok {
			e.documents.nextAccess++
			doc.identity.accessedAt = e.documents.nextAccess
			doc.edits.changedSinceAccess = false
		}
	}
}
