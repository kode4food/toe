package defaults

import (
	"runtime"

	"github.com/kode4food/toe/internal/view"
)

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func intOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func lineNumberOr(
	v view.LineNumber, def view.LineNumber,
) view.LineNumber {
	if v == "" {
		return def
	}
	return v
}

func bufferLineOr(
	v view.BufferLine, def view.BufferLine,
) view.BufferLine {
	if v == "" {
		return def
	}
	return v
}

func defaultShell() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C"}
	}
	return []string{"sh", "-c"}
}
