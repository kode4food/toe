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

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	imageRegistry struct {
		placed       map[uint32][2]int
		ready        map[uint32][2]int
		placeholders map[[2]int][]string
		graphics     bool
		remote       bool
	}

	imageReadyMsg struct {
		id   uint32
		size [2]int
	}

	imageTransmitMsg struct {
		raw  string
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

	// Assumed cell pixel size; only caps the transmit byte budget since kitty
	// refits into the cell box. ponytail: query CSI 14 t if it ever blurs
	cellPixelW = 10
	cellPixelH = 20
)

func newImageRegistry() *imageRegistry {
	return &imageRegistry{
		placed:       map[uint32][2]int{},
		ready:        map[uint32][2]int{},
		placeholders: map[[2]int][]string{},
		graphics:     graphicsSupported(),
		remote:       isRemoteSession(),
	}
}

type displayArgs struct {
	img        *Image
	path       string
	id         uint32
	cols, rows int
}

func (r *imageRegistry) display(a displayArgs) tea.Cmd {
	if !r.graphics {
		return nil
	}
	size := [2]int{a.cols, a.rows}
	placed, transmitted := r.placed[a.id]
	if transmitted && placed == size {
		return nil
	}
	r.preparePlaceholders(size)
	r.placed[a.id] = size

	// Encode off the event loop: scaling a large image can take 100s of ms.
	// Deleting the old placement first refits the terminal to the new box
	remote := r.remote
	return func() tea.Msg {
		var buf bytes.Buffer
		writeKittyDelete(&buf, a.id)
		if !transmitted {
			err := transmit(transmitArgs{
				buf:    &buf,
				img:    a.img,
				path:   a.path,
				id:     a.id,
				cols:   a.cols,
				rows:   a.rows,
				remote: remote,
			})
			if err != nil {
				return nil
			}
		} else {
			opts := &kitty.Options{
				Action: kitty.Put, Quiet: 2, ID: int(a.id),
				Columns: a.cols, Rows: a.rows, VirtualPlacement: true,
			}
			buf.WriteString(ansi.KittyGraphics(nil, opts.Options()...))
		}
		return imageTransmitMsg{raw: buf.String(), id: a.id, size: size}
	}
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
		cmds = append(cmds, m.context.images.display(displayArgs{
			img:  img,
			path: pane.Path(),
			id:   id,
			cols: cols,
			rows: rows,
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
	buf        *bytes.Buffer
	img        *Image
	path       string
	id         uint32
	cols, rows int
	remote     bool
}

// transmit encodes a fresh image into buf via the cheapest medium: a local PNG
// is read off disk, other local sources use a temp file, SSH streams the pixels
func transmit(args transmitArgs) error {
	opts := &kitty.Options{
		Action:           kitty.TransmitAndPut,
		Format:           kitty.PNG,
		Quiet:            2,
		ID:               int(args.id),
		Columns:          args.cols,
		Rows:             args.rows,
		VirtualPlacement: true,
	}
	switch {
	case args.remote:
		opts.Transmission, opts.Chunk = kitty.Direct, true
		img := scaleForCells(args.img, args.cols, args.rows)
		return kitty.EncodeGraphics(args.buf, img, opts)
	case args.img.format == "png":
		// the terminal reads the PNG off disk and scales it, so encode nothing
		opts.Transmission, opts.File = kitty.File, args.path
		return kitty.EncodeGraphics(args.buf, nil, opts)
	default:
		opts.Transmission = kitty.TempFile
		img := scaleForCells(args.img, args.cols, args.rows)
		return kitty.EncodeGraphics(args.buf, img, opts)
	}
}

func scaleForCells(img image.Image, cols, rows int) image.Image {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	tw, th := cols*cellPixelW, rows*cellPixelH
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
