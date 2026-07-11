package vcs

import (
	"slices"
	"sync"
	"time"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Differ tracks the hunks between a version-control base text and a live
	// document text, recomputing on a background goroutine. Updates are
	// debounced so rapid keystrokes coalesce into one diff
	Differ struct {
		mu     sync.RWMutex
		base   core.Rope
		doc    core.Rope
		hunks  []view.DiffHunk
		events chan differEvent
		done   chan struct{}
		notify func()
	}

	differEvent struct {
		text   core.Rope
		isBase bool
	}
)

// diffDebounce is how long the worker waits for further updates before
// recomputing; the gutter trails a keystroke by this much
const diffDebounce = 50 * time.Millisecond

// NewDiffer starts a differ for the given base and document text. notify is
// invoked from the worker goroutine after every recompute
func NewDiffer(base, doc core.Rope, notify func()) *Differ {
	d := &Differ{
		base:   base,
		doc:    doc,
		events: make(chan differEvent, 64),
		done:   make(chan struct{}),
		notify: notify,
	}
	go d.run()
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
	d.send(differEvent{text: doc})
}

// SetBase schedules a recompute against a new diff base text
func (d *Differ) SetBase(base core.Rope) {
	d.send(differEvent{text: base, isBase: true})
}

// Close stops the worker goroutine. The differ must not be used after
func (d *Differ) Close() {
	close(d.done)
}

func (d *Differ) send(ev differEvent) {
	select {
	case d.events <- ev:
	case <-d.done:
	}
}

func (d *Differ) run() {
	d.recompute()
	for {
		select {
		case <-d.done:
			return
		case ev := <-d.events:
			d.apply(ev)
			if !d.settle() {
				return
			}
			d.recompute()
		}
	}
}

// settle drains further events until the debounce window passes without a new
// one. It reports false when the differ was closed
func (d *Differ) settle() bool {
	timer := time.NewTimer(diffDebounce)
	defer timer.Stop()
	for {
		select {
		case <-d.done:
			return false
		case ev := <-d.events:
			d.apply(ev)
			timer.Reset(diffDebounce)
		case <-timer.C:
			return true
		}
	}
}

func (d *Differ) apply(ev differEvent) {
	d.mu.Lock()
	if ev.isBase {
		d.base = ev.text
	} else {
		d.doc = ev.text
	}
	d.mu.Unlock()
}

func (d *Differ) recompute() {
	d.mu.RLock()
	base, doc := d.base, d.doc
	d.mu.RUnlock()
	hunks := Diff(base, doc)
	d.mu.Lock()
	d.hunks = hunks
	d.mu.Unlock()
	if d.notify != nil {
		d.notify()
	}
}
