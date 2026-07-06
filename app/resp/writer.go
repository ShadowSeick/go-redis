package resp

import (
	"bufio"
	"io"
)

// This is going to be simple, this needs to respond with the values that should be written in the output.
// \r\n, lengths, etc
type Writer struct {
	wt *bufio.Writer
}

func NewWriter(w io.Writer) Writer {
	return Writer{
		wt: bufio.NewWriterSize(w, DefaultBufferSize),
	}
}

func (w Writer) WriteType(reply byte) {
	w.wt.WriteByte(reply)
}

func (w Writer) WriteReply(reply []byte) {
	w.wt.Write(reply)
	w.writeFinal()
}

func (w Writer) writeFinal() {
	w.wt.Write([]byte("\r\n"))
}
