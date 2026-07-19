package ui

import (
	"bytes"
	"image"
	"math"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	"golang.org/x/image/draw"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	imageRegistry struct {
		placed       map[uint32]geom.Size
		ready        map[uint32]geom.Size
		placeholders map[geom.Size][]string
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

	// imagePlacementID: fixed so each reput replaces the placement in place,
	// never accumulating placements, so a resize needs no delete first
	imagePlacementID = 1

	// Assumed cell pixel size; only caps the transmit byte budget since kitty
	// refits into the cell box. ponytail: query CSI 14 t if it ever blurs
	cellPixelW = 10
	cellPixelH = 20
)

func newImageRegistry() *imageRegistry {
	return &imageRegistry{
		placed:       map[uint32]geom.Size{},
		ready:        map[uint32]geom.Size{},
		placeholders: map[geom.Size][]string{},
		graphics:     graphicsSupported(),
		remote:       isRemoteSession(),
	}
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
	placed, transmitted := r.placed[a.id]
	if transmitted && placed == size {
		return nil
	}
	r.preparePlaceholders(size)
	r.placed[a.id] = size

	// Encode off the event loop: scaling a large image can take 100s of ms.
	// The reput reuses imagePlacementID, so the terminal refits the existing
	// placement in place without a delete-then-put blink
	remote := r.remote
	return func() tea.Msg {
		var buf bytes.Buffer
		if !transmitted {
			err := transmit(transmitArgs{
				buf:    &buf,
				img:    a.img,
				path:   a.path,
				id:     a.id,
				cells:  a.cells,
				remote: remote,
			})
			if err != nil {
				return nil
			}
		} else {
			opts := &kitty.Options{
				Action:           kitty.Put,
				Quiet:            2,
				ID:               int(a.id),
				PlacementID:      imagePlacementID,
				Columns:          a.cells.Width,
				Rows:             a.cells.Height,
				VirtualPlacement: true,
			}
			buf.WriteString(ansi.KittyGraphics(nil, opts.Options()...))
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

// transmit encodes a fresh image into buf via the cheapest medium: a local PNG
// is read off disk, other local sources use a temp file, SSH streams the pixels
func transmit(args transmitArgs) error {
	opts := &kitty.Options{
		Action:           kitty.TransmitAndPut,
		Format:           kitty.PNG,
		Quiet:            2,
		ID:               int(args.id),
		PlacementID:      imagePlacementID,
		Columns:          args.cells.Width,
		Rows:             args.cells.Height,
		VirtualPlacement: true,
	}
	switch {
	case args.remote:
		opts.Transmission, opts.Chunk = kitty.Direct, true
		img := scaleForCells(args.img, args.cells)
		return kitty.EncodeGraphics(args.buf, img, opts)
	case args.img.format == "png":
		// the terminal reads the PNG off disk and scales it, so encode nothing
		opts.Transmission, opts.File = kitty.File, args.path
		return kitty.EncodeGraphics(args.buf, nil, opts)
	default:
		opts.Transmission = kitty.TempFile
		img := scaleForCells(args.img, args.cells)
		return kitty.EncodeGraphics(args.buf, img, opts)
	}
}

func scaleForCells(img image.Image, cells geom.Size) image.Image {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	tw, th := cells.Width*cellPixelW, cells.Height*cellPixelH
	if sw <= tw && sh <= th {
		return img
	}
	scale := math.Min(float64(tw)/float64(sw), float64(th)/float64(sh))
	dw := max(int(float64(sw)*scale), 1)
	dh := max(int(float64(sh)*scale), 1)
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	// ApproxBiLinear: CatmullRom dominated transmit time for no visible gain
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
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
