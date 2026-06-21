package core

import "io"

// RopeReader streams rope contents through io.Reader
type RopeReader struct {
	text []byte
	pos  int
}

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
