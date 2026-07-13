package resp

import (
	"bufio"
	"io"
)

type Writer struct {
	wt *bufio.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		wt: bufio.NewWriterSize(w, DefaultBufferSize),
	}
}

// This needs to be changed. I don't like the way I am delegating this into the caller of the writer
func (w Writer) WriteType(reply byte) error {
	return w.wt.WriteByte(reply)
}

func (w Writer) WriteReply(reply []byte) error {
	if _, err := w.wt.Write(reply); err != nil {
		return err
	}
	return w.crlf()
}

func (w Writer) crlf() error {
	_, err := w.wt.Write(crlf)
	if err != nil {
		return err
	}

	return nil
}

func (w Writer) Flush() error {
	return w.wt.Flush()
}
