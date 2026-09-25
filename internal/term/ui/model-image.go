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
		placed       map[uint32]imageState
		ready        map[uint32]imageState
		sent         map[uint32]bool
		used         map[uint32]int
		placeholders map[geom.Size][]string
		cell         geom.Size
		settleBy     time.Time
		frame        int
		graphics     bool
		remote       bool
	}

	imageReadyMsg struct {
		id    uint32
		state imageState
	}

	imageTransmitMsg struct {
		raw   string
		id    uint32
		state imageState
	}

	imageState struct {
		cells geom.Size
		rev   uint64
	}
)

const (
	imageIDMask               = 0x7FFFFF
	previewImageMask          = 0x800000
	imagePlacementIDMask      = 0xFFFFFF
	imageViewSalt             = 0x9E3779
	imageCellAspect           = 2
	kittyQuietNoResponse byte = 2

	// imageTransmitDelay lets Bubble Tea enter the alternate screen before
	// Kitty receives image data that the screen transition can discard
	imageTransmitDelay = 40 * time.Millisecond

	// Images shown this frame cannot be evicted, so the cap is soft
	maxResidentImages = 24

	settleWait = 250 * time.Millisecond
)

func newImageRegistry() *imageRegistry {
	return &imageRegistry{
		placed:       map[uint32]imageState{},
		ready:        map[uint32]imageState{},
		sent:         map[uint32]bool{},
		used:         map[uint32]int{},
		placeholders: map[geom.Size][]string{},
		graphics:     graphicsSupported(),
		settleBy:     time.Now().Add(settleWait),
		remote:       isRemoteSession(),
	}
}

func (r *imageRegistry) settling() bool {
	return r.graphics && time.Now().Before(r.settleBy)
}

func (r *imageRegistry) setCell(size geom.Size) bool {
	if r.cell == size || size.IsEmpty() {
		return false
	}
	r.cell = size
	return true
}

func (r *imageRegistry) beginFrame() {
	r.frame++
}

func (r *imageRegistry) inFlight(id uint32) bool {
	placed, ok := r.placed[id]
	return ok && r.ready[id] != placed
}

func (r *imageRegistry) evict(keep uint32) string {
	var buf bytes.Buffer
	for len(r.placed) > maxResidentImages {
		var victim uint32
		oldest := r.frame
		found := false
		for id, used := range r.used {
			if id == keep || used >= r.frame {
				continue
			}
			if !found || used < oldest {
				victim = id
				oldest = used
				found = true
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
	rev   uint64
}

func (r *imageRegistry) display(a displayArgs) tea.Cmd {
	if !r.graphics {
		return nil
	}
	state := imageState{cells: a.cells, rev: a.rev}
	r.used[a.id] = r.frame
	if r.ready[a.id] == state {
		r.placed[a.id] = state
		return nil
	}
	if r.placed[a.id] == state {
		// already requested this exact state, so wait for its response rather
		// than queuing a duplicate
		return nil
	}
	// an unsent initial transmit is still in flight, so a put now would
	// reference an id the terminal doesn't have yet and get dropped
	_, everEmitted := r.placed[a.id]
	if everEmitted && !r.sent[a.id] {
		return nil
	}
	r.preparePlaceholders(state.cells)
	resize := r.placed[a.id].cells != state.cells
	r.placed[a.id] = state
	// a put re-places pixels the terminal holds, never new ones
	if r.sent[a.id] && resize {
		put := putSeq(a.id, state.cells)
		return func() tea.Msg {
			return imageTransmitMsg{raw: put, id: a.id, state: state}
		}
	}
	evict := r.evict(a.id)
	remote := r.remote
	first := !r.sent[a.id]
	return func() tea.Msg {
		if first {
			time.Sleep(imageTransmitDelay)
		}
		var buf bytes.Buffer
		buf.WriteString(evict)
		if err := transmit(transmitArgs{
			buf:    &buf,
			img:    a.img,
			path:   a.path,
			id:     a.id,
			cells:  state.cells,
			remote: remote,
		}); err != nil {
			return nil
		}
		return imageTransmitMsg{raw: buf.String(), id: a.id, state: state}
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
	return r.ready[id].cells == cells
}

func (r *imageRegistry) readySize(id uint32) (geom.Size, bool) {
	state, ok := r.ready[id]
	return state.cells, ok
}

func (m Model) imageDisplayCmd() tea.Cmd {
	cx := m.context
	var cmds []tea.Cmd
	cx.Editor.Tree().RangeVisible(func(p view.Pane) bool {
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
		if cells.IsEmpty() {
			return true
		}
		id := kittyImageID(kittyImageIDArgs{
			content: img.ContentID(),
			surface: uint32(pane.ID()),
		})
		cmds = append(cmds, cx.images.display(displayArgs{
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

func transmit(args transmitArgs) error {
	opts := &kitty.Options{
		Action:           kitty.TransmitAndPut,
		Format:           kitty.PNG,
		Quiet:            kittyQuietNoResponse,
		ID:               int(args.id),
		PlacementID:      int(imagePlacementID(args.id)),
		Columns:          args.cells.Width,
		Rows:             args.cells.Height,
		VirtualPlacement: true,
	}
	switch {
	case args.remote || args.path == "":
		opts.Transmission = kitty.Direct
		opts.Chunk = true
		return kitty.EncodeGraphics(args.buf, args.img.Image, opts)
	case args.img.format == "png":
		opts.Transmission = kitty.File
		opts.File = args.path
		return kitty.EncodeGraphics(args.buf, nil, opts)
	default:
		opts.Transmission = kitty.TempFile
		return kitty.EncodeGraphics(args.buf, args.img.Image, opts)
	}
}

func putSeq(id uint32, cells geom.Size) string {
	opts := &kitty.Options{
		Action:           kitty.Put,
		Quiet:            kittyQuietNoResponse,
		ID:               int(id),
		PlacementID:      int(imagePlacementID(id)),
		Columns:          cells.Width,
		Rows:             cells.Height,
		VirtualPlacement: true,
	}
	return ansi.KittyGraphics(nil, opts.Options()...)
}

func imagePlacementID(id uint32) uint32 {
	return max((id+1)&imagePlacementIDMask, 1)
}

func deleteImageSeq(id uint32) string {
	opts := &kitty.Options{
		Action:          kitty.Delete,
		Delete:          kitty.DeleteID,
		ID:              int(id),
		DeleteResources: true,
		Quiet:           kittyQuietNoResponse,
	}
	return ansi.KittyGraphics(nil, opts.Options()...)
}

type kittyImageIDArgs struct {
	content uint32
	surface uint32
	preview bool
}

func kittyImageID(args kittyImageIDArgs) uint32 {
	id := (args.content ^ args.surface*imageViewSalt) & imageIDMask
	if id == 0 {
		id = 1
	}
	if args.preview {
		id |= previewImageMask
	}
	return id
}

type imageCellSizeArgs struct {
	maxCells geom.Size
	pixels   geom.Size
}

func imageCellSize(args imageCellSizeArgs) geom.Size {
	if args.maxCells.IsEmpty() || args.pixels.IsEmpty() {
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
