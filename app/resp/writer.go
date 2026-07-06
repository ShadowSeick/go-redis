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

// This should encapsulate the right writing. It should only have WriteString
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
	_, err := w.wt.Write(carrierReturn)
	if err != nil {
		return err
	}

	return nil
}

func (w Writer) Flush() error {
	return w.wt.Flush()
}
