package lsp

import (
	"sync"

	"github.com/kode4food/toe/internal/view"
)

// docState tracks, per document, which servers are attached, whether didOpen
// has been sent (or is still pending an in-flight open), and the last
// pull-diagnostics result ID per provider
type docState struct {
	sync.RWMutex
	serverNames map[view.DocumentId][]string
	opened      map[view.DocumentId]bool
	pendingOpen map[view.DocumentId]bool
	diagIDs     map[view.DocumentId]map[string]string
}

func (d *docState) previousDiagnosticID(
	id view.DocumentId, provider string,
) *string {
	d.RLock()
	defer d.RUnlock()
	ids, ok := d.diagIDs[id]
	if !ok {
		return nil
	}
	prev, ok := ids[provider]
	if !ok {
		return nil
	}
	return &prev
}

func (d *docState) setPreviousDiagnosticID(
	id view.DocumentId, provider string, resultID *string,
) {
	d.Lock()
	defer d.Unlock()
	if resultID == nil {
		if ids, ok := d.diagIDs[id]; ok {
			delete(ids, provider)
		}
		return
	}
	ids, ok := d.diagIDs[id]
	if !ok {
		ids = map[string]string{}
		d.diagIDs[id] = ids
	}
	ids[provider] = *resultID
}

func (d *docState) setServerNames(id view.DocumentId, names []string) {
	d.Lock()
	defer d.Unlock()
	d.serverNames[id] = names
}

// claimOpen reports whether the caller is the first to open id, marking it
// opened as a side effect
func (d *docState) claimOpen(id view.DocumentId) bool {
	d.Lock()
	defer d.Unlock()
	if d.opened[id] {
		return false
	}
	d.opened[id] = true
	return true
}

func (d *docState) markPendingOpen(docs []*view.Document) {
	d.Lock()
	defer d.Unlock()
	for _, doc := range docs {
		d.pendingOpen[doc.ID()] = true
	}
}

// consumePendingOpen reports whether id still needs its startup open; false
// means the document was closed before this call ran
func (d *docState) consumePendingOpen(id view.DocumentId) bool {
	d.Lock()
	defer d.Unlock()
	want := d.pendingOpen[id]
	delete(d.pendingOpen, id)
	return want
}

// cancelPendingOpen cancels any in-flight startup open for id and reports
// whether didOpen was already sent for it
func (d *docState) cancelPendingOpen(id view.DocumentId) bool {
	d.Lock()
	defer d.Unlock()
	opened := d.opened[id]
	delete(d.pendingOpen, id)
	return opened
}

// forget drops all bookkeeping for a closed document
func (d *docState) forget(id view.DocumentId) {
	d.Lock()
	defer d.Unlock()
	delete(d.serverNames, id)
	delete(d.opened, id)
	delete(d.diagIDs, id)
}

func (d *docState) reset() {
	d.Lock()
	defer d.Unlock()
	d.serverNames = map[view.DocumentId][]string{}
	d.opened = map[view.DocumentId]bool{}
	d.pendingOpen = map[view.DocumentId]bool{}
	d.diagIDs = map[view.DocumentId]map[string]string{}
}
