package ansibump

import (
	"bytes"
	"io"
)

// Deprecated: NewDecoder is deprecated as it was unsafe.
// Use [NewDecoder] instead.
func (c *Customizer) NewDecoder() *Decoder {
	d := NewDecoder(nil)
	return d
}

// Deprecated: Buffer is deprecated as it was not idiomatic.
// Use [Customizer.Bytes] instead.
func (c *Customizer) Buffer(r io.Reader) (*bytes.Buffer, error) {
	buf, err := c.Bytes(r)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(buf), nil
}

// Deprecated: Read reads bytes from r and interprets ANSI sequences, updating the buffer.
// Use [Decoder.Decode] instead.
func (d *Decoder) Read(r io.Reader) error {
	return d.Decode(r)
}
