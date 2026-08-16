package view

// LastModifiedDocIDs returns the two most recently modified-and-left documents,
// with the most recent first. Invalid entries have value InvalidDocumentId
func (e *Editor) LastModifiedDocIDs() [2]DocumentId {
	return e.documents.lastModifiedIDs
}

// PopPrevDocID returns and removes the most recently accessed document for the
// focused view
func (e *Editor) PopPrevDocID() (DocumentId, bool) {
	v := e.FocusedView()
	if v == nil {
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

func (e *Editor) recordPrevDoc() {
	if v := e.FocusedView(); v != nil {
		v.addDocHistory(v.DocID())
		e.recordLeavingDocFor(v)
	}
}

func (e *Editor) recordLeavingDoc() {
	if v := e.FocusedView(); v != nil {
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
	if v := e.FocusedView(); v != nil {
		if doc, ok := e.documents.byID[v.DocID()]; ok {
			e.documents.nextAccess++
			doc.identity.accessedAt = e.documents.nextAccess
			doc.edits.changedSinceAccess = false
		}
	}
}
