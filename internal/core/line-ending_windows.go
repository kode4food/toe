//go:build windows

package core

// NativeLineEnding is the platform's default line ending
func NativeLineEnding() LineEnding {
	return LineEndingCRLF
}
