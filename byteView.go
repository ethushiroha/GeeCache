package GeeCache

type ByteView struct {
	b []byte
}

func (b ByteView) Len() int {
	return len(b.b)
}

func (b ByteView) String() string {
	return string(b.b)
}

func (b ByteView) ByteSlice() []byte {
	return clone(b.b)
}

func clone(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
