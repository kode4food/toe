package vcs_test

import (
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
)

const differBurst = 100

func TestDiffer(t *testing.T) {
	t.Run("computes initial hunks", func(t *testing.T) {
		base := core.NewRope("a\nb\n")
		doc := core.NewRope("a\nx\n")
		updated := make(chan struct{}, 16)
		d := vcs.NewDiffer(view.DiffSides{
			Base: base,
			Doc:  doc,
		}, func() { updated <- struct{}{} })
		defer d.Close()

		waitUpdate(t, updated)
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
		}, d.Hunks())
		assert.Equal(t, "a\nb\n", d.Base().String())
	})

	t.Run("recomputes after document update", func(t *testing.T) {
		base := core.NewRope("a\nb\n")
		updated := make(chan struct{}, 16)
		d := vcs.NewDiffer(view.DiffSides{
			Base: base,
			Doc:  base,
		}, func() { updated <- struct{}{} })
		defer d.Close()

		waitUpdate(t, updated)
		assert.Empty(t, d.Hunks())

		d.SetDoc(core.NewRope("a\nchanged\n"))
		waitUpdate(t, updated)
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
		}, d.Hunks())
	})

	t.Run("recomputes after base update", func(t *testing.T) {
		doc := core.NewRope("a\nb\n")
		updated := make(chan struct{}, 16)
		d := vcs.NewDiffer(view.DiffSides{
			Base: doc,
			Doc:  doc,
		}, func() { updated <- struct{}{} })
		defer d.Close()

		waitUpdate(t, updated)
		d.SetBase(core.NewRope("a\n"))
		waitUpdate(t, updated)
		assert.Len(t, d.Hunks(), 1)
	})

	t.Run("close stops updates", func(t *testing.T) {
		doc := core.NewRope("a\n")
		updated := make(chan struct{}, 16)
		d := vcs.NewDiffer(view.DiffSides{
			Base: doc,
			Doc:  doc,
		}, func() { updated <- struct{}{} })
		waitUpdate(t, updated)
		d.Close()
		// sends after close must not block or panic
		d.SetDoc(core.NewRope("b\n"))
	})

	t.Run("coalesces bursts", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var updates atomic.Int32
			base := core.NewRope("base\n")
			d := vcs.NewDiffer(
				view.DiffSides{
					Base: base,
					Doc:  base,
				},
				func() {
					updates.Add(1)
				},
			)
			defer d.Close()
			synctest.Wait()
			assert.EqualValues(t, 1, updates.Load())
			for range differBurst {
				d.SetDoc(core.NewRope("changed\n"))
			}
			d.SetDoc(base)
			time.Sleep(time.Second)
			synctest.Wait()
			assert.EqualValues(t, 2, updates.Load())
			assert.Empty(t, d.Hunks())
		})
	})
}

func waitUpdate(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for diff update")
	}
}
