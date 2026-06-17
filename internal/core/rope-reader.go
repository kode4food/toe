package core

import "io"

type (
	// RopeReader streams rope contents through io.Reader
	RopeReader struct {
		text []byte
		pos  int
	}
)

func NewRopeReader(r Rope) *RopeReader {
	return &RopeReader{text: []byte(r.String())}
}

func (r *RopeReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.text) {
		return 0, io.EOF
	}
	n := copy(p, r.text[r.pos:])
	r.pos += n
	return n, nil
}
