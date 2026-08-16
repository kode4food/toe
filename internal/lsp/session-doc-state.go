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
	if ids, ok := d.diagIDs[id]; ok {
		if prev, ok := ids[provider]; ok {
			return &prev
		}
	}
	return nil
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

func (d *docState) consumePendingOpen(id view.DocumentId) bool {
	d.Lock()
	defer d.Unlock()
	want := d.pendingOpen[id]
	delete(d.pendingOpen, id)
	return want
}

func (d *docState) cancelPendingOpen(id view.DocumentId) bool {
	d.Lock()
	defer d.Unlock()
	opened := d.opened[id]
	delete(d.pendingOpen, id)
	return opened
}

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
