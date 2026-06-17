//go:build windows

package core

func NativeLineEnding() LineEnding { return LineEndingCRLF }
