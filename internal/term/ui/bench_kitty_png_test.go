package ui_test

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/ui"
)

// BenchmarkKittyPNGEncoding compares wrapped and direct RGBA PNG encoding
func BenchmarkKittyPNGEncoding(b *testing.B) {
	for _, size := range []struct {
		name string
		area geom.Size
	}{
		{name: "scrollbar", area: geom.Size{Width: 9, Height: 456}},
		{name: "large", area: geom.Size{Width: 400, Height: 300}},
	} {
		pixels := image.NewRGBA(
			image.Rect(0, 0, size.area.Width, size.area.Height),
		)
		for y := range size.area.Height {
			shade := uint8(y * 7)
			for x := range size.area.Width {
				pixels.SetRGBA(x, y, color.RGBA{
					R: shade,
					G: shade / 2,
					B: 255 - shade,
					A: 255,
				})
			}
		}
		wrapped := &ui.Image{Image: pixels}
		for _, variant := range []struct {
			name string
			img  image.Image
		}{
			{name: "wrapped", img: wrapped},
			{name: "rgba", img: pixels},
		} {
			b.Run(size.name+"/"+variant.name, func(b *testing.B) {
				var buf bytes.Buffer
				opts := &kitty.Options{
					Action:       kitty.TransmitAndPut,
					Format:       kitty.PNG,
					Transmission: kitty.Direct,
					Chunk:        true,
				}
				b.ReportAllocs()
				for b.Loop() {
					buf.Reset()
					if err := kitty.EncodeGraphics(
						&buf, variant.img, opts,
					); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
