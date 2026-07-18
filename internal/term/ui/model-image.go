package ui

import (
	"bytes"
	"image"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	imageRegistry struct {
		placed       map[uint32][2]int
		ready        map[uint32][2]int
		placeholders map[[2]int][]string
		graphics     bool
	}

	imageReadyMsg struct {
		id   uint32
		size [2]int
	}

	imageDisplayRequestMsg struct {
		gen int
	}
)

const (
	imageIDMask       = 0x7FFFFF
	previewImageMask  = 0x800000
	imageViewSalt     = 0x9E3779
	imageCellAspect   = 2
	imageDisplayDelay = 40 * time.Millisecond
)

func newImageRegistry() *imageRegistry {
	return &imageRegistry{
		placed:       map[uint32][2]int{},
		ready:        map[uint32][2]int{},
		placeholders: map[[2]int][]string{},
		graphics:     graphicsSupported(),
	}
}

func (r *imageRegistry) display(
	id uint32, img image.Image, cols, rows int,
) tea.Cmd {
	if !r.graphics {
		return nil
	}
	size := [2]int{cols, rows}
	placed, transmitted := r.placed[id]
	if transmitted && placed == size {
		return nil
	}
	r.preparePlaceholders(size)

	// A fresh image is transmitted and placed in one command; a resize/zoom
	// only re-places the already-transmitted image. Either way the old
	// placement is deleted first so the terminal refits to the new cell box
	var buf bytes.Buffer
	writeKittyDelete(&buf, id)
	if !transmitted {
		opts := &kitty.Options{
			Action: kitty.TransmitAndPut, Format: kitty.PNG,
			Quiet: 2, ID: int(id), Columns: cols, Rows: rows,
			VirtualPlacement: true, Transmission: kitty.Direct,
			Chunk: true,
		}
		if err := kitty.EncodeGraphics(&buf, img, opts); err != nil {
			return nil
		}
	} else {
		opts := &kitty.Options{
			Action: kitty.Put, Quiet: 2, ID: int(id),
			Columns: cols, Rows: rows, VirtualPlacement: true,
		}
		buf.WriteString(ansi.KittyGraphics(nil, opts.Options()...))
	}
	r.placed[id] = size
	return tea.Sequence(tea.Raw(buf.String()), func() tea.Msg {
		return imageReadyMsg{id: id, size: size}
	})
}

func (r *imageRegistry) placeholder(cols, rows, row, col int) string {
	return r.placeholders[[2]int{cols, rows}][row*cols+col]
}

func (r *imageRegistry) preparePlaceholders(size [2]int) {
	if _, ok := r.placeholders[size]; ok {
		return
	}
	cols, rows := size[0], size[1]
	cells := make([]string, cols*rows)
	for row := range rows {
		for col := range cols {
			cells[row*cols+col] = tui.PlaceholderSymbol(row, col)
		}
	}
	r.placeholders[size] = cells
}

func (r *imageRegistry) isReady(id uint32, cols, rows int) bool {
	return r.ready[id] == [2]int{cols, rows}
}

func (r *imageRegistry) readySize(id uint32) ([2]int, bool) {
	size, ok := r.ready[id]
	return size, ok
}

func (m Model) imageDisplayCmd() tea.Cmd {
	var cmds []tea.Cmd
	m.context.Editor.Tree().Range(func(p view.Pane) bool {
		pane, ok := p.(*ImagePane)
		if !ok {
			return true
		}
		img := pane.Image()
		w, h := img.Size()
		a := pane.Area()
		cols, rows := imagePaneCellSize(
			pane, a.Width, max(a.Height-1, 0), w, h,
		)
		if cols == 0 || rows == 0 {
			return true
		}
		id := kittyImageID(img.ContentID(), uint32(pane.ID()), false)
		cmds = append(cmds, m.context.images.display(id, img, cols, rows))
		return true
	})
	if len(cmds) == 0 {
		return nil
	}
	return tea.Sequence(cmds...)
}

func kittyImageID(content, surface uint32, preview bool) uint32 {
	id := (content ^ surface*imageViewSalt) & imageIDMask
	if id == 0 {
		id = 1
	}
	if preview {
		id |= previewImageMask
	}
	return id
}

func imageCellSize(maxCols, maxRows, imgW, imgH int) (int, int) {
	if maxCols <= 0 || maxRows <= 0 || imgW <= 0 || imgH <= 0 {
		return 0, 0
	}
	cols := maxCols
	ratio := float64(imgH) / (float64(imgW) * imageCellAspect)
	rows := max(int(float64(cols)*ratio), 1)
	if rows > maxRows {
		rows = maxRows
		cols = max(int(float64(rows)/ratio), 1)
	}
	return cols, rows
}

func writeKittyDelete(buf *bytes.Buffer, id uint32) {
	opts := &kitty.Options{
		Action: kitty.Delete, Quiet: 2, ID: int(id),
		Delete: kitty.DeleteID, DeleteResources: false,
	}
	buf.WriteString(ansi.KittyGraphics(nil, opts.Options()...))
}
