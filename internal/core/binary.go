package core

import (
	"bytes"
	"unicode/utf8"
)

type textEncoding int

const (
	encUTF8 textEncoding = iota
	encUTF16LE
	encUTF16BE
	encUTF32LE
	encUTF32BE
)

const (
	binarySampleSize = 32 * 1024

	// maxControlRatio applies once the sample is confirmed valid UTF-8, a
	// strong text signal on its own; legacyControlRatio is stricter since
	// invalid UTF-8 (a legacy 8-bit encoding, or binary) gets no such signal
	maxControlRatio    = 0.30
	legacyControlRatio = 0.10
)

var binarySignatures = [][]byte{
	{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, // PNG
	{0xFF, 0xD8, 0xFF},                            // JPEG
	{'G', 'I', 'F', '8', '7', 'a'},                // GIF87a
	{'G', 'I', 'F', '8', '9', 'a'},                // GIF89a
	{'%', 'P', 'D', 'F', '-'},                     // PDF
	{'P', 'K', 0x03, 0x04},                        // ZIP/JAR/DOCX/...
	{0x1f, 0x8b, 0x08},                            // GZIP
	{0x7f, 'E', 'L', 'F'},                         // ELF
	{0xFE, 0xED, 0xFA, 0xCE},                      // Mach-O 32 BE
	{0xFE, 0xED, 0xFA, 0xCF},                      // Mach-O 64 BE
	{0xCE, 0xFA, 0xED, 0xFE},                      // Mach-O 32 LE
	{0xCF, 0xFA, 0xED, 0xFE},                      // Mach-O 64 LE
}

// LooksBinary reports whether data appears to be non-text content, biased
// toward false negatives: garbled binary is recoverable, a refused-open text
// file is not
func LooksBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data[:min(len(data), binarySampleSize)]

	if hasBinarySignature(sample) {
		return true
	}
	if enc, prefix, ok := detectBOM(sample); ok {
		return !validEncodedText(sample[prefix:], enc)
	}
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}
	threshold := legacyControlRatio
	if utf8.Valid(sample) {
		threshold = maxControlRatio
	}
	return controlCharRatio(sample) > threshold
}

func hasBinarySignature(sample []byte) bool {
	for _, sig := range binarySignatures {
		if bytes.HasPrefix(sample, sig) {
			return true
		}
	}
	return false
}

// detectBOM matches the longest prefix first, since the 2-byte UTF-16LE BOM
// (FF FE) is itself a prefix of the 4-byte UTF-32LE BOM
func detectBOM(sample []byte) (enc textEncoding, prefixLen int, ok bool) {
	switch {
	case bytes.HasPrefix(sample, []byte{0xFF, 0xFE, 0x00, 0x00}):
		return encUTF32LE, 4, true
	case bytes.HasPrefix(sample, []byte{0x00, 0x00, 0xFE, 0xFF}):
		return encUTF32BE, 4, true
	case bytes.HasPrefix(sample, []byte{0xEF, 0xBB, 0xBF}):
		return encUTF8, 3, true
	case bytes.HasPrefix(sample, []byte{0xFF, 0xFE}):
		return encUTF16LE, 2, true
	case bytes.HasPrefix(sample, []byte{0xFE, 0xFF}):
		return encUTF16BE, 2, true
	default:
		return 0, 0, false
	}
}

func validEncodedText(rest []byte, enc textEncoding) bool {
	switch enc {
	case encUTF8:
		return utf8.Valid(rest)
	case encUTF16LE, encUTF16BE:
		return validUTF16(rest, enc == encUTF16BE)
	default:
		return validUTF32(rest, enc == encUTF32BE)
	}
}

// validUTF16 structurally validates surrogate pairing, ignoring a trailing odd
// byte left by sample truncation
func validUTF16(rest []byte, bigEndian bool) bool {
	pendingHighSurrogate := false
	for i := 0; i+1 < len(rest); i += 2 {
		var unit uint16
		if bigEndian {
			unit = uint16(rest[i])<<8 | uint16(rest[i+1])
		} else {
			unit = uint16(rest[i+1])<<8 | uint16(rest[i])
		}
		isHigh := unit >= 0xD800 && unit <= 0xDBFF
		isLow := unit >= 0xDC00 && unit <= 0xDFFF
		switch {
		case pendingHighSurrogate && !isLow:
			return false
		case pendingHighSurrogate:
			pendingHighSurrogate = false
		case isLow:
			return false
		case isHigh:
			pendingHighSurrogate = true
		}
	}
	return !pendingHighSurrogate
}

// validUTF32 checks each code point is in range and not a surrogate, ignoring a
// trailing partial unit left by sample truncation
func validUTF32(rest []byte, bigEndian bool) bool {
	for i := 0; i+3 < len(rest); i += 4 {
		var cp uint32
		if bigEndian {
			cp = uint32(rest[i])<<24 | uint32(rest[i+1])<<16 |
				uint32(rest[i+2])<<8 | uint32(rest[i+3])
		} else {
			cp = uint32(rest[i+3])<<24 | uint32(rest[i+2])<<16 |
				uint32(rest[i+1])<<8 | uint32(rest[i])
		}
		if cp > 0x10FFFF || (cp >= 0xD800 && cp <= 0xDFFF) {
			return false
		}
	}
	return true
}

func controlCharRatio(sample []byte) float64 {
	var controls int
	for _, b := range sample {
		switch {
		case b == '\t' || b == '\n' || b == '\r' || b == '\f':
			// normal textual whitespace, not a binary signal
		case b < 0x20 || b == 0x7f:
			controls++
		}
	}
	return float64(controls) / float64(len(sample))
}
