package vcs

import (
	"slices"
	"sync"
	"time"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/debounce"
	"github.com/kode4food/toe/internal/view"
)

// Differ tracks the hunks between a version-control base text and a live document text. Updates are
// debounced so rapid keystrokes coalesce into one diff
type Differ struct {
	mu         sync.RWMutex
	base       core.Rope
	doc        core.Rope
	hunks      []view.DiffHunk
	gen        uint64
	closed     bool
	recomputes *debounce.Keyed[struct{}]
	notify     func()
}

// diffDebounce is how long a differ waits for further updates before recomputing. The gutter trails
// a keystroke by this much
const diffDebounce = 50 * time.Millisecond

// NewDiffer starts a differ for the given base and document text. notify is invoked from a
// background goroutine after every recompute
func NewDiffer(sides view.DiffSides, notify func()) *Differ {
	d := &Differ{
		base:       sides.Base,
		doc:        sides.Doc,
		recomputes: debounce.NewKeyed[struct{}](diffDebounce),
		notify:     notify,
	}
	go d.recompute()
	return d
}

// Hunks returns a snapshot of the current hunks
func (d *Differ) Hunks() []view.DiffHunk {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return slices.Clone(d.hunks)
}

// Base returns the current diff base text
func (d *Differ) Base() core.Rope {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.base
}

// SetDoc schedules a recompute against a new document text
func (d *Differ) SetDoc(doc core.Rope) {
	d.mu.Lock()
	d.doc = doc
	d.gen++
	d.mu.Unlock()
	d.recomputes.Do(struct{}{}, d.recompute)
}

// SetBase schedules a recompute against a new diff base text
func (d *Differ) SetBase(base core.Rope) {
	d.mu.Lock()
	d.base = base
	d.gen++
	d.mu.Unlock()
	d.recomputes.Do(struct{}{}, d.recompute)
}

// Close drops pending recomputes and stops publishing hunks
func (d *Differ) Close() {
	d.mu.Lock()
	d.closed = true
	d.mu.Unlock()
	d.recomputes.Stop()
}

func (d *Differ) recompute() {
	d.mu.RLock()
	sides := view.DiffSides{
		Base: d.base,
		Doc:  d.doc,
	}
	gen := d.gen
	d.mu.RUnlock()
	hunks := view.Diff(sides)
	d.mu.Lock()
	// A slow recompute may finish after a newer one; only the newest publishes
	if d.closed || gen != d.gen {
		d.mu.Unlock()
		return
	}
	d.hunks = hunks
	d.mu.Unlock()
	if d.notify != nil {
		d.notify()
	}
}
