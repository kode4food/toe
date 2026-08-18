package core_test

import (
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/core"
)

func BenchmarkReflowHardWrap(b *testing.B) {
	ascii := strings.Repeat("the quick brown fox jumps over a dog ", 40)
	wide := strings.Repeat("速い茶色の狐が怠惰な犬を飛び越える ", 40)
	// e + U+0301, the case a per-rune width walk measures as two cells
	combining := strings.Repeat("été élégant ", 80)

	for _, bc := range []struct {
		name string
		text string
	}{
		{"ascii", ascii},
		{"wide", wide},
		{"combining", combining},
	} {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				core.ReflowHardWrap(bc.text, 72)
			}
		})
	}
}
