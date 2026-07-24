package ui

import (
	"bytes"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	imageRegistry struct {
		placed       map[uint32]geom.Size
		ready        map[uint32]geom.Size
		used         map[uint32]int
		placeholders map[geom.Size][]string
		frame        int
		graphics     bool
		remote       bool
	}

	imageReadyMsg struct {
		id   uint32
		size geom.Size
	}

	imageTransmitMsg struct {
		raw  string
		id   uint32
		size geom.Size
	}
)

const (
	imageIDMask        = 0x7FFFFF
	previewImageMask   = 0x800000
	imageViewSalt      = 0x9E3779
	imageCellAspect    = 2
	imageTransmitDelay = 40 * time.Millisecond

	// Placement IDs use a 24-bit underline color, split across both dimensions
	imagePlacementDimensionBits = 12
	imagePlacementDimensionMask = 1<<imagePlacementDimensionBits - 1

	// maxResidentImages caps images kept resident in the terminal. Soft: one
	// shown this frame is never evicted, so the real ceiling is what fits
	maxResidentImages = 24
)

func newImageRegistry() *imageRegistry {
	return &imageRegistry{
		placed:       map[uint32]geom.Size{},
		ready:        map[uint32]geom.Size{},
		used:         map[uint32]int{},
		placeholders: map[geom.Size][]string{},
		graphics:     graphicsSupported(),
		remote:       isRemoteSession(),
	}
}

func (r *imageRegistry) beginFrame() {
	r.frame++
}

func (r *imageRegistry) inFlight(id uint32) bool {
	placed, ok := r.placed[id]
	return ok && r.ready[id] != placed
}

// evict removes stale images, excluding keep and this frame's images, and
// returns their terminal deletion sequences
func (r *imageRegistry) evict(keep uint32) string {
	var buf bytes.Buffer
	for len(r.placed) > maxResidentImages {
		victim, oldest, found := uint32(0), r.frame, false
		for id, used := range r.used {
			if id == keep || used >= r.frame {
				continue
			}
			if !found || used < oldest {
				victim, oldest, found = id, used, true
			}
		}
		if !found {
			break
		}
		buf.WriteString(deleteImageSeq(victim))
		delete(r.placed, victim)
		delete(r.ready, victim)
		delete(r.used, victim)
	}
	return buf.String()
}

type displayArgs struct {
	img   *Image
	path  string
	id    uint32
	cells geom.Size
}

func (r *imageRegistry) display(a displayArgs) tea.Cmd {
	if !r.graphics {
		return nil
	}
	size := a.cells
	r.used[a.id] = r.frame
	placed, transmitted := r.placed[a.id]
	if transmitted && r.ready[a.id] == size {
		r.placed[a.id] = size
		return nil
	}
	if transmitted && placed == size {
		return nil
	}
	if transmitted {
		if _, ready := r.ready[a.id]; !ready {
			return nil
		}
		r.preparePlaceholders(size)
		r.placed[a.id] = size
		put := putSeq(a.id, size)
		return func() tea.Msg {
			return imageTransmitMsg{raw: put, id: a.id, size: size}
		}
	}
	r.preparePlaceholders(size)
	r.placed[a.id] = size
	evict := r.evict(a.id)
	remote := r.remote
	return func() tea.Msg {
		// Let Bubble Tea enter the alternate screen before Kitty receives image
		// data that the screen transition can discard
		time.Sleep(imageTransmitDelay)
		var buf bytes.Buffer
		buf.WriteString(evict)
		if err := transmit(transmitArgs{
			buf:    &buf,
			img:    a.img,
			path:   a.path,
			id:     a.id,
			cells:  size,
			remote: remote,
		}); err != nil {
			return nil
		}
		return imageTransmitMsg{raw: buf.String(), id: a.id, size: size}
	}
}

func (r *imageRegistry) placeholder(cells geom.Size, at geom.Point) string {
	return r.placeholders[cells][at.Y*cells.Width+at.X]
}

func (r *imageRegistry) preparePlaceholders(cells geom.Size) {
	if _, ok := r.placeholders[cells]; ok {
		return
	}
	placeholders := make([]string, cells.Width*cells.Height)
	for row := range cells.Height {
		for col := range cells.Width {
			at := geom.Point{X: col, Y: row}
			placeholders[row*cells.Width+col] = tui.PlaceholderSymbol(at)
		}
	}
	r.placeholders[cells] = placeholders
}

func (r *imageRegistry) isReady(id uint32, cells geom.Size) bool {
	return r.ready[id] == cells
}

func (r *imageRegistry) readySize(id uint32) (geom.Size, bool) {
	cells, ok := r.ready[id]
	return cells, ok
}

func (m Model) imageDisplayCmd() tea.Cmd {
	var cmds []tea.Cmd
	m.context.Editor.Tree().Range(func(p view.Pane) bool {
		pane, ok := p.(*ImagePane)
		if !ok {
			return true
		}
		img := pane.Image()
		pixels := img.Size()
		a := pane.Area()
		cells := imagePaneCellSize(imagePaneCellSizeArgs{
			pane: pane,
			maxCells: geom.Size{
				Width:  a.Width,
				Height: max(a.Height-1, 0),
			},
			pixels: pixels,
		})
		if cells.Empty() {
			return true
		}
		id := kittyImageID(img.ContentID(), uint32(pane.ID()), false)
		cmds = append(cmds, m.context.images.display(displayArgs{
			img:   img,
			path:  pane.Path(),
			id:    id,
			cells: cells,
		}))
		return true
	})
	if len(cmds) == 0 {
		return nil
	}
	return tea.Sequence(cmds...)
}

func isRemoteSession() bool {
	return os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != ""
}

type transmitArgs struct {
	buf    *bytes.Buffer
	img    *Image
	path   string
	id     uint32
	cells  geom.Size
	remote bool
}

// transmit sends full-resolution pixels through the cheapest medium; Kitty
// handles scaling
func transmit(args transmitArgs) error {
	opts := &kitty.Options{
		Action:           kitty.TransmitAndPut,
		Format:           kitty.PNG,
		Quiet:            2,
		ID:               int(args.id),
		PlacementID:      int(imagePlacementID(args.cells)),
		Columns:          args.cells.Width,
		Rows:             args.cells.Height,
		VirtualPlacement: true,
	}
	switch {
	case args.remote:
		opts.Transmission, opts.Chunk = kitty.Direct, true
		return kitty.EncodeGraphics(args.buf, args.img, opts)
	case args.img.format == "png":
		opts.Transmission, opts.File = kitty.File, args.path
		return kitty.EncodeGraphics(args.buf, nil, opts)
	default:
		opts.Transmission = kitty.TempFile
		return kitty.EncodeGraphics(args.buf, args.img, opts)
	}
}

// putSeq places a resident image on a new cell grid without sending pixels
func putSeq(id uint32, cells geom.Size) string {
	opts := &kitty.Options{
		Action:           kitty.Put,
		Quiet:            2,
		ID:               int(id),
		PlacementID:      int(imagePlacementID(cells)),
		Columns:          cells.Width,
		Rows:             cells.Height,
		VirtualPlacement: true,
	}
	return ansi.KittyGraphics(nil, opts.Options()...)
}

func deleteImageSeq(id uint32) string {
	opts := &kitty.Options{
		Action:          kitty.Delete,
		Delete:          kitty.DeleteID,
		ID:              int(id),
		DeleteResources: true,
		Quiet:           2,
	}
	return ansi.KittyGraphics(nil, opts.Options()...)
}

func imagePlacementID(cells geom.Size) uint32 {
	width := uint32(cells.Width) & imagePlacementDimensionMask
	height := uint32(cells.Height) & imagePlacementDimensionMask
	return max(width<<imagePlacementDimensionBits|height, 1)
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

type imageCellSizeArgs struct {
	maxCells geom.Size
	pixels   geom.Size
}

func imageCellSize(args imageCellSizeArgs) geom.Size {
	if args.maxCells.Empty() || args.pixels.Empty() {
		return geom.Size{}
	}
	cols := args.maxCells.Width
	ratio := float64(args.pixels.Height) /
		(float64(args.pixels.Width) * imageCellAspect)
	rows := max(int(float64(cols)*ratio), 1)
	if rows > args.maxCells.Height {
		rows = args.maxCells.Height
		cols = max(int(float64(rows)/ratio), 1)
	}
	return geom.Size{Width: cols, Height: rows}
}
