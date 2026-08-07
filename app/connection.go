package main

import (
	"bytes"
	"fmt"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/logging"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

type Conn struct {
	conn   net.Conn
	logger logging.Logger
	reader *resp.Reader
	writer *resp.Writer
}

func NewConnection(conn net.Conn, logger logging.Logger, memory datastructures.LRU) *Conn {
	return &Conn{
		conn:   conn,
		logger: logger,
		reader: resp.NewReader(conn),
		writer: resp.NewWriter(conn),
	}
}

func (c *Conn) Peek(n int) ([]byte, error) {
	return c.reader.Peek(n)
}

func (c *Conn) Close() {
	c.conn.Close()
}

func (c *Conn) ReadReply() (any, error) {
	return c.reader.ReadReply()
}

func (c *Conn) Flush() error {
	return c.writer.Flush()
}

func (c *Conn) WriteStatus(status []byte) error {
	if err := c.writer.WriteType(resp.RespTypeStatus); err != nil {
		return fmt.Errorf("error writting status type")
	}
	if err := c.writer.WriteReply(status); err != nil {
		return fmt.Errorf("error writting status reply, %w", err)
	}
	return nil
}

func (c *Conn) WriteString(value []byte) error {
	if err := c.writer.WriteType(resp.RespTypeString); err != nil {
		return fmt.Errorf("error writting string type, %w", err)
	}
	if !bytes.Equal(value, []byte(commands.NullString)) {
		if err := c.WriteLength(len(value)); err != nil {
			return fmt.Errorf("error writting string length, %w", err)
		}
	}
	if err := c.writer.WriteReply(value); err != nil {
		return fmt.Errorf("error writting string reply, %w", err)
	}
	return nil
}

func (c *Conn) WriteInteger(value []byte) error {
	if err := c.writer.WriteType(resp.RespTypeInt); err != nil {
		return fmt.Errorf("error writting int type, %w", err)
	}
	if err := c.writer.WriteReply([]byte(value)); err != nil {
		return fmt.Errorf("error writting int reply, %w", err)
	}
	return nil
}

func (c *Conn) WriteArray(values []string) error {
	if err := c.writer.WriteType(resp.RespTypeArray); err != nil {
		return fmt.Errorf("error writting array type, %w", err)
	}
	if err := c.WriteLength(len(values)); err != nil {
		return fmt.Errorf("error writting array length, %w", err)
	}
	for _, val := range values {
		if err := c.WriteString([]byte(val)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) WriteLength(length int) error {
	val := strconv.Itoa(length)
	if err := c.writer.WriteReply([]byte(val)); err != nil {
		return err
	}
	return nil
}
