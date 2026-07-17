package view

import "github.com/kode4food/toe/internal/core"

// insertAccum holds the pre-insert state and the composed changeset for the
// current insert-mode session
type insertAccum struct {
	oldState core.State
	cs       core.ChangeSet
}

// BeginInsertGroup starts insert-mode change accumulation for vid if not
// already active. Subsequent Apply calls accumulate into a single history
// revision until CommitInsertHistory is called
func (d *Document) BeginInsertGroup(vid Id) {
	if d.buf.insertAcc != nil {
		return
	}
	d.buf.insertAcc = &insertAccum{
		oldState: core.State{
			Doc:       d.buf.text,
			Selection: d.SelectionFor(vid),
		},
		cs: core.NewChangeSet(d.buf.text),
	}
}

// CommitInsertHistory flushes any accumulated insert-mode changes as one
// history revision. It is a no-op when no accumulation is active
func (d *Document) CommitInsertHistory(vid Id) {
	acc := d.buf.insertAcc
	d.buf.insertAcc = nil
	if acc == nil || acc.cs.Empty() {
		return
	}
	tx := core.NewTransaction(acc.oldState.Doc).
		WithChanges(acc.cs).
		WithSelection(d.SelectionFor(vid))
	_ = d.buf.history.CommitRevision(tx, acc.oldState)
}

// Apply applies a transaction to the document, recording in history. While an
// insert group is active (BeginInsertGroup was called), changes are accumulated
// and a single revision is committed by CommitInsertHistory
func (d *Document) Apply(tx core.Transaction, vid Id) error {
	d.ensureLoaded()
	newText, err := tx.Apply(d.buf.text)
	if err != nil {
		return err
	}

	if d.buf.insertAcc != nil {
		cs := tx.Changes()
		newSel := d.resolveAppliedSelection(vid, tx, cs)
		oldSel := d.buf.selections[vid]
		d.buf.Lock()
		d.buf.text = newText
		d.buf.selections[vid] = newSel
		if !cs.Empty() {
			d.buf.insertAcc.cs = d.buf.insertAcc.cs.Compose(cs)
			d.mapOtherSelections(vid, cs)
			d.buf.unsaved = true
			d.buf.modified = true
			d.buf.version++
		}
		d.buf.Unlock()
		if !cs.Empty() {
			d.remapOverlays(cs)
			d.markAllDirty()
		} else if !oldSel.Equal(newSel) {
			d.markDirty(vid)
		}
		return nil
	}

	cs := tx.Changes()
	newSel := d.resolveAppliedSelection(vid, tx, cs)
	oldSel := d.buf.selections[vid]
	if !cs.Empty() {
		// Commit the FORWARD tx with the BEFORE state so Undo can restore it
		beforeSt := core.State{Doc: d.buf.text, Selection: d.SelectionFor(vid)}
		if err := d.buf.history.CommitRevision(tx, beforeSt); err != nil {
			return err
		}
	}
	d.buf.Lock()
	d.buf.text = newText
	d.buf.selections[vid] = newSel
	if !cs.Empty() {
		d.mapOtherSelections(vid, cs)
		d.buf.unsaved = true
		d.buf.modified = true
		d.buf.version++
	}
	d.buf.Unlock()
	if !cs.Empty() {
		d.remapOverlays(cs)
		d.markAllDirty()
	} else if !oldSel.Equal(newSel) {
		d.markDirty(vid)
	}
	return nil
}

// Undo reverts one history step for the given view
func (d *Document) Undo(vid Id) bool {
	tx, ok := d.buf.history.Undo()
	if !ok {
		return false
	}
	newText, err := tx.Apply(d.buf.text)
	if err != nil {
		return false
	}
	cs := tx.Changes()
	d.buf.Lock()
	if !cs.Empty() {
		d.mapOtherSelections(vid, cs)
	}
	d.buf.text = newText
	d.buf.version++
	if txSel := tx.Selection(); txSel != nil {
		d.buf.selections[vid] = *txSel
	}
	d.buf.unsaved = d.Modified()
	d.buf.modified = true
	d.buf.Unlock()
	d.markAllDirty()
	if !cs.Empty() {
		d.remapOverlays(cs)
	}
	return true
}

// Redo reapplies one reverted step for the given view
func (d *Document) Redo(vid Id) bool {
	tx, ok := d.buf.history.Redo()
	if !ok {
		return false
	}
	newText, err := tx.Apply(d.buf.text)
	if err != nil {
		return false
	}
	cs := tx.Changes()
	d.buf.Lock()
	if !cs.Empty() {
		d.mapOtherSelections(vid, cs)
	}
	d.buf.text = newText
	d.buf.version++
	if txSel := tx.Selection(); txSel != nil {
		d.buf.selections[vid] = *txSel
	}
	d.buf.unsaved = d.Modified()
	d.buf.modified = true
	d.buf.Unlock()
	d.markAllDirty()
	if !cs.Empty() {
		d.remapOverlays(cs)
	}
	return true
}

func (d *Document) resolveAppliedSelection(
	vid Id, tx core.Transaction, cs core.ChangeSet,
) core.Selection {
	if txSel := tx.Selection(); txSel != nil {
		return *txSel
	}
	cur := d.SelectionFor(vid)
	if mapped, err := cur.Map(cs); err == nil {
		return mapped
	}
	return cur
}

func (d *Document) mapOtherSelections(vid Id, cs core.ChangeSet) {
	for otherVid, sel := range d.buf.selections {
		if otherVid == vid {
			continue
		}
		if mapped, err := sel.Map(cs); err == nil {
			d.buf.selections[otherVid] = mapped
		}
	}
}
