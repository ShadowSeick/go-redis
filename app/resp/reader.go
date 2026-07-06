package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var (
	ErrRespTypeNotImplemented = errors.New("data type not implemented")
	ErrNotValid               = errors.New("input not valid")
)

// redis resp protocol data type.
const (
	RespTypeStatus    = '+' // +<string>\r\n
	RespTypeError     = '-' // -<string>\r\n
	RespTypeString    = '$' // $<length>\r\n<bytes>\r\n
	RespTypeInt       = ':' // :<number>\r\n
	RespTypeNil       = '_' // _\r\n
	RespTypeFloat     = ',' // ,<floating-point-number>\r\n (golang float)
	RespTypeBool      = '#' // true: #t\r\n false: #f\r\n
	RespTypeBlobError = '!' // !<length>\r\n<bytes>\r\n
	RespTypeVerbatim  = '=' // =<length>\r\nFORMAT:<bytes>\r\n
	RespTypeBigInt    = '(' // (<big number>\r\n
	RespTypeArray     = '*' // *<len>\r\n... (same as resp2)
	RespTypeMap       = '%' // %<len>\r\n(key)\r\n(value)\r\n... (golang map)
	RespTypeSet       = '~' // ~<len>\r\n... (same as Array)
	RespTypeAttr      = '|' // |<len>\r\n(key)\r\n(value)\r\n... + command reply
	RespTypePush      = '>' // ><len>\r\n... (same as Array)
)

// DefaultBufferSize is the default size for read/write buffers (32 KiB).
const DefaultBufferSize = 32 * 1024

type Reader struct {
	rd *bufio.Reader
}

func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd: bufio.NewReaderSize(rd, DefaultBufferSize),
	}
}

func (r *Reader) Buffered() int {
	return r.rd.Buffered()
}

func (r *Reader) Peek(n int) ([]byte, error) {
	return r.rd.Peek(n)
}

func (r *Reader) Discard(n int) (int, error) {
	return r.rd.Discard(n)
}

func (r *Reader) ReadReply() (any, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}

	switch line[0] {
	case RespTypeStatus:
		return string(line[1:]), nil
	case RespTypeString:
		return r.readString(line)
	case RespTypeArray:
		return r.readArray(line)
	default:
		return nil, ErrRespTypeNotImplemented
	}
}

func (r *Reader) readLine() ([]byte, error) {
	line, err := r.rd.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// I know it's valid when the \r\n is valid one
	if line[len(line)-1] != '\n' && line[len(line)-2] != '\r' {
		return nil, ErrNotValid
	}
	fmt.Println(string(line))

	return line[:len(line)-2], nil
}

func (r *Reader) readString(line []byte) (string, error) {
	size, err := readLen(line)
	if err != nil {
		return "", err
	}

	str := make([]byte, size+2)
	n, err := io.ReadFull(r.rd, str)
	if err != nil && err != io.EOF {
		return "", err
	}

	if int(size) != n {
		return "", errors.New("the actual string was shorter than expected")
	}

	return string(str[:len(str)-2]), nil
}

func (r *Reader) readArray(line []byte) ([]any, error) {
	size, err := readLen(line)
	if err != nil {
		return nil, err
	}

	values := make([]any, size)
	for i := range size {
		v, err := r.ReadReply()
		if err != nil {
			return nil, err
		}
		values[i] = v
	}

	return values, nil
}

func readLen(line []byte) (int, error) {
	n, err := strconv.Atoi(string(line[1:]))
	if err != nil {
		return 0, err
	}

	return n, nil
}
