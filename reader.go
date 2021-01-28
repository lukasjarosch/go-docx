package docx

import "io"

// Reader is a very basic io.Reader implementation which is capable of returning the current position.
type Reader struct {
	str      string
	i        int64
	length   int64
	prevRune int64 // index of the previously read rune or -1
}

// NewReader returns a new Reader given a string source.
func NewReader(s string) *Reader {
	return &Reader{
		str:      s,
		i:        0,
		length:   int64(len(s)),
		prevRune: -1,
	}
}

// String implements the Stringer interface.
func (r *Reader) String() string {
	return r.str
}

// Len returns the current length of the stream which has been read.
func (r *Reader) Len() int {
	if r.i >= r.length {
		return 0
	}
	return int(r.length - r.i)
}

// Size returns the size of the string to read.
func (r *Reader) Size() int64 {
	return r.length
}

// Pos returns the current position which the reader is at.
func (r *Reader) Pos() int64 {
	return r.i
}

// Read implements the io.Reader interface.
func (r *Reader) Read(b []byte) (int, error) {
	if r.i >= r.length {
		return 0, io.EOF
	}

	r.prevRune = -1
	b[0] = r.str[r.i]
	r.i += 1
	return 1, nil
}

// ReadByte implements hte io.ByteReader interface.
func (r *Reader) ReadByte() (byte, error) {
	r.prevRune = -1
	if r.i >= int64(len(r.str)) {
		return 0, io.EOF
	}
	b := r.str[r.i]
	r.i++
	return b, nil
}
