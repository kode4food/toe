package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
)

// TestImageDisplayNoStarvation guards a resize held during an in-flight
// initial transmit against being dropped instead of retried once sent
func TestImageDisplayNoStarvation(t *testing.T) {
	r := newImageRegistry()
	r.graphics = true
	img := &Image{id: 1, format: "png"}
	small := geom.Size{Width: 2, Height: 2}
	size := geom.Size{Width: 100, Height: 50}

	// first request (pre-layout, degenerate size) is in flight, unconfirmed
	cmd1 := r.display(displayArgs{img: img, path: "x", id: 7, cells: small})
	assert.NotNil(t, cmd1, "first request must transmit")

	// a second, different size arrives before the first is confirmed sent —
	// must be held, not turned into a premature put
	cmd2 := r.display(displayArgs{img: img, path: "x", id: 7, cells: size})
	assert.Nil(t, cmd2, "must not put before the initial transmit is sent")
	assert.Equal(t, small, r.placed[7],
		"the unconfirmed size must not be overwritten while suppressed")

	// the initial transmit lands
	r.sent[7] = true

	// the held request must now go through
	cmd3 := r.display(displayArgs{img: img, path: "x", id: 7, cells: size})
	assert.NotNil(t, cmd3, "the suppressed size must be retried, not lost")
	assert.Equal(t, size, r.placed[7])
}
