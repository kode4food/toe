package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type binaryRowStyles struct {
	location tui.Style
	hex      tui.Style
	chars    tui.Style
	border   tui.Style
}

const (
	binaryGroupBytes     = 8
	binaryGroupWidth     = binaryGroupBytes*4 + 1
	binaryMinOffsetWidth = 8
	binaryRowPadding     = 4
	binaryTargetGroups   = 2
)

func (r *renderPass) renderBinaryPane(
	buf *tui.Buffer, pane *BinaryPane, y0 int, focused bool,
) {
	a := pane.Area()
	content := geom.Area{
		Point: geom.Point{X: a.X, Y: y0 + a.Y},
		Size:  geom.Size{Width: a.Width, Height: max(a.Height-1, 0)},
	}
	th := r.cx.ThemeFor(focused)
	style := th.Get("ui.text")
	data, err := pane.readVisible()
	if err != nil {
		renderCenteredMessage(buf, content, i18n.ErrorText(err), style)
	} else {
		styles := binaryStyles(th, style)
		width := pane.bytesPerRow()
		for row := range content.Height {
			start := row * width
			if start >= len(data) {
				break
			}
			end := min(start+width, len(data))
			renderBinaryRow(renderBinaryRowArgs{
				buf:         buf,
				at:          content.Point.Add(geom.Point{Y: row}),
				offset:      pane.offset + int64(start),
				data:        data[start:end],
				bytesPerRow: width,
				offsetWidth: pane.offsetWidth(),
				maxWidth:    content.Width,
				styles:      styles,
			})
		}
	}
	r.renderBinaryStatus(buf, pane, y0, focused)
}

func (r *renderPass) renderBinaryStatus(
	buf *tui.Buffer, pane *BinaryPane, y0 int, focused bool,
) {
	a := pane.Area()
	th := r.cx.Theme()
	baseTUI := th.Get("ui.statusline.inactive")
	modeSt := baseTUI
	if focused {
		baseTUI = th.Get("ui.statusline")
		modeSt = th.Get("ui.statusline." + pane.Mode().Scope())
	}
	name := view.DocumentRelativeName(pane.path, r.cx.Editor.Cwd())
	right := []statusElem{{
		text:  fmt.Sprintf("%d / %d bytes", pane.offset, pane.size),
		style: baseTUI,
	}}
	right = r.withMaximizedStatus(right)
	renderStatusElems(renderStatusElemsArgs{
		buf:       buf,
		at:        geom.Point{X: a.X, Y: y0 + a.Bottom()},
		width:     a.Width,
		baseStyle: baseTUI,
		left: []statusElem{
			statusBadge(pane.Mode().String(), modeSt),
			{text: name, style: baseTUI},
		},
		right: right,
	})
}

type renderBinaryRowArgs struct {
	buf         *tui.Buffer
	at          geom.Point
	offset      int64
	data        []byte
	bytesPerRow int
	offsetWidth int
	maxWidth    int
	styles      binaryRowStyles
}

func renderBinaryRow(args renderBinaryRowArgs) {
	x, remaining := args.at.X, args.maxWidth
	write := func(text string, style tui.Style) {
		if remaining <= 0 {
			return
		}
		text = runewidth.Truncate(text, remaining, "")
		args.buf.SetString(geom.Point{X: x, Y: args.at.Y}, text, style)
		width := runewidth.StringWidth(text)
		x += width
		remaining -= width
	}
	write(
		fmt.Sprintf("%0*x  ", args.offsetWidth, args.offset),
		args.styles.location,
	)
	var hex strings.Builder
	for i := range args.bytesPerRow {
		if i > 0 && i%binaryGroupBytes == 0 {
			hex.WriteByte(' ')
		}
		if i < len(args.data) {
			_, _ = fmt.Fprintf(&hex, "%02x ", args.data[i])
		} else {
			hex.WriteString("   ")
		}
	}
	write(hex.String()+" ", args.styles.hex)
	write(borderV, args.styles.border)
	var chars strings.Builder
	for _, b := range args.data {
		if b >= 0x20 && b <= 0x7e {
			chars.WriteByte(b)
		} else {
			chars.WriteByte('.')
		}
	}
	write(chars.String(), args.styles.chars)
	write(borderV, args.styles.border)
}

func binaryStyles(th *theme.Theme, base tui.Style) binaryRowStyles {
	return binaryRowStyles{
		location: th.Get("ui.linenr"),
		hex:      applyAccentStyle(base, th.Get("constant")),
		chars:    applyAccentStyle(base, th.Get("string")),
		border:   applyAccentStyle(base, th.Get("ui.border")),
	}
}

type binaryBytesPerRowArgs struct {
	width       int
	offsetWidth int
}

func binaryBytesPerRow(args binaryBytesPerRowArgs) int {
	groups := max(
		(args.width-args.offsetWidth-binaryRowPadding)/binaryGroupWidth, 1,
	)
	return groups * binaryGroupBytes
}

func binaryTargetWidth(offsetWidth int) int {
	return offsetWidth + binaryRowPadding + binaryTargetGroups*binaryGroupWidth
}

func binaryOffsetWidth(size int64) int {
	width := 1
	for n := size; n >= 16; n /= 16 {
		width++
	}
	return max(width, binaryMinOffsetWidth)
}
