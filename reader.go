package docx

import "io"

type Reader struct {
	s        string
	i        int64
	z        int64
	prevRune int64 // index of the previously read rune or -1
}

func NewReader(s string) *Reader {
	return &Reader{
		s:        s,
		i:        0,
		z:        int64(len(s)),
		prevRune: -1,
	}
}

func (r *Reader) String() string {
	return r.s
}

func (r *Reader) Len() int {
	if r.i >= r.z {
		return 0
	}
	return int(r.z - r.i)
}


func (r *Reader) Size() int64 {
	return r.z
}


func (r *Reader) Pos() int64 {
	return r.i
}

func (r *Reader) Read(b []byte) (int, error) {
	if r.i >= r.z {
		return 0, io.EOF
	}

	r.prevRune = -1
	b[0] = r.s[r.i]
	r.i += 1
	return 1, nil
}

func (r *Reader) ReadByte() (byte, error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	b := r.s[r.i]
	r.i++
	return b, nil
}
