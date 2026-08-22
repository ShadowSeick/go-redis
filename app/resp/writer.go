package resp

import (
	"bufio"
	"errors"
	"io"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/pkg/util"
)

var ErrTypeNotImplemented = errors.New("type not implemented")

const numBuffSize = 11 // 8 bytes + 3 bytes for integer type and crlf.

type Writer struct {
	wt      *bufio.Writer
	numBuff []byte
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		wt:      bufio.NewWriterSize(w, DefaultBufferSize),
		numBuff: make([]byte, 0, numBuffSize),
	}
}

func (w Writer) Reset(wt io.Writer) {
	w.wt.Reset(wt)
}

func (w Writer) Flush() error {
	return w.wt.Flush()
}

func (w Writer) Write(reply any) (err error) {
	switch v := reply.(type) {
	case byte:
		err = w.wt.WriteByte(v)
	case []byte:
		_, err = w.wt.Write(v)
	case rune:
		_, err = w.wt.WriteRune(v)
	case string:
		err = w.string(v)
	case int:
		err = w.integer(v)
	default:
		err = ErrTypeNotImplemented
	}
	return err
}

func (w Writer) string(s string) error {
	b := util.StringToByte(s)
	if _, err := w.wt.Write(b); err != nil {
		return err
	}
	return w.crlf()
}

func (w Writer) integer(n int) error {
	b := strconv.AppendInt(w.numBuff[:0], int64(n), 10)
	if _, err := w.wt.Write(b); err != nil {
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
