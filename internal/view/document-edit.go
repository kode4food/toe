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
	if d.edits.insertAcc != nil {
		return
	}
	d.edits.insertAcc = &insertAccum{
		oldState: core.State{
			Doc:       d.content.text,
			Selection: d.SelectionFor(vid),
		},
		cs: core.NewChangeSet(d.content.text),
	}
}

// CommitInsertHistory flushes any accumulated insert-mode changes as one
// history revision. It is a no-op when no accumulation is active
func (d *Document) CommitInsertHistory(vid Id) {
	acc := d.edits.insertAcc
	d.edits.insertAcc = nil
	if acc == nil || acc.cs.Empty() {
		return
	}
	tx := core.NewTransaction(acc.oldState.Doc).
		WithChanges(acc.cs).
		WithSelection(d.SelectionFor(vid))
	_ = d.edits.history.CommitRevision(tx, acc.oldState)
}

// Apply applies a transaction to the document, recording in history. While an
// insert group is active (BeginInsertGroup was called), changes are accumulated
// and a single revision is committed by CommitInsertHistory
func (d *Document) Apply(tx core.Transaction, vid Id) error {
	d.ensureLoaded()
	newText, err := tx.Apply(d.content.text)
	if err != nil {
		return err
	}

	if d.edits.insertAcc != nil {
		cs := tx.Changes()
		newSel := d.resolveAppliedSelection(vid, tx, cs)
		oldSel := d.views.selections[vid]
		d.content.Lock()
		d.content.text = newText
		if !cs.Empty() {
			d.content.version++
		}
		d.content.Unlock()
		d.views.selections[vid] = newSel
		if !cs.Empty() {
			d.edits.insertAcc.cs = d.edits.insertAcc.cs.Compose(cs)
			d.edits.changedSinceAccess = true
			d.mapOtherSelections(vid, cs)
			d.remapOverlays(cs)
			d.MarkDirty()
		} else if !oldSel.Equal(newSel) {
			d.markViewDirty(vid)
		}
		return nil
	}

	cs := tx.Changes()
	newSel := d.resolveAppliedSelection(vid, tx, cs)
	oldSel := d.views.selections[vid]
	if !cs.Empty() {
		// Commit the FORWARD tx with the BEFORE state so Undo can restore it
		beforeSt := core.State{
			Doc:       d.content.text,
			Selection: d.SelectionFor(vid),
		}
		if err := d.edits.history.CommitRevision(tx, beforeSt); err != nil {
			return err
		}
	}
	d.content.Lock()
	d.content.text = newText
	if !cs.Empty() {
		d.content.version++
	}
	d.content.Unlock()
	d.views.selections[vid] = newSel
	if !cs.Empty() {
		d.edits.changedSinceAccess = true
		d.mapOtherSelections(vid, cs)
		d.remapOverlays(cs)
		d.MarkDirty()
	} else if !oldSel.Equal(newSel) {
		d.markViewDirty(vid)
	}
	return nil
}

// Undo reverts one history step for the given view
func (d *Document) Undo(vid Id) bool {
	tx, ok := d.edits.history.Undo()
	if !ok {
		return false
	}
	newText, err := tx.Apply(d.content.text)
	if err != nil {
		return false
	}
	cs := tx.Changes()
	if !cs.Empty() {
		d.mapOtherSelections(vid, cs)
	}
	d.content.Lock()
	d.content.text = newText
	d.content.version++
	d.content.Unlock()
	if txSel := tx.Selection(); txSel != nil {
		d.views.selections[vid] = *txSel
	}
	d.edits.changedSinceAccess = true
	d.MarkDirty()
	if !cs.Empty() {
		d.remapOverlays(cs)
	}
	return true
}

// Redo reapplies one reverted step for the given view
func (d *Document) Redo(vid Id) bool {
	tx, ok := d.edits.history.Redo()
	if !ok {
		return false
	}
	newText, err := tx.Apply(d.content.text)
	if err != nil {
		return false
	}
	cs := tx.Changes()
	if !cs.Empty() {
		d.mapOtherSelections(vid, cs)
	}
	d.content.Lock()
	d.content.text = newText
	d.content.version++
	d.content.Unlock()
	if txSel := tx.Selection(); txSel != nil {
		d.views.selections[vid] = *txSel
	}
	d.edits.changedSinceAccess = true
	d.MarkDirty()
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
	for otherVid, sel := range d.views.selections {
		if otherVid == vid {
			continue
		}
		if mapped, err := sel.Map(cs); err == nil {
			d.views.selections[otherVid] = mapped
		}
	}
}
