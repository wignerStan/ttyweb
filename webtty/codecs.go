package webtty

// Decoder decodes binary data from src into dst.
type Decoder interface {
	Decode(dst, src []byte) (int, error)
}

// Encoder encodes binary data from src into dst.
type Encoder interface {
	Encode(dst, src []byte) (int, error)
}

// NullCodec is a pass-through codec that copies data without encoding.
type NullCodec struct{}

// Encode copies src into dst without encoding.
func (NullCodec) Encode(dst, src []byte) (int, error) {
	return copy(dst, src), nil
}

// Decode copies src into dst without decoding.
func (NullCodec) Decode(dst, src []byte) (int, error) {
	return copy(dst, src), nil
}
