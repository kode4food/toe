//go:build !windows

package core

func NativeLineEnding() LineEnding {
	return LineEndingLF
}
