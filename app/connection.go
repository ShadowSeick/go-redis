package main

import (
	"fmt"
	"net"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

type Conn struct {
	conn   net.Conn
	reader *resp.Reader
	writer *resp.Writer
}

func NewConnection(conn net.Conn, memory datastructures.LRU) *Conn {
	return &Conn{
		conn:   conn,
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

func (c *Conn) WriteStatus(status string) error {
	if err := c.writer.Write(resp.RespTypeStatus); err != nil {
		return fmt.Errorf("error writting status type, %w", err)
	}
	if err := c.writer.Write(status); err != nil {
		return fmt.Errorf("error writting status reply, %w", err)
	}
	return nil
}

func (c *Conn) WriteError(msg string) error {
	if err := c.writer.Write(resp.RespTypeError); err != nil {
		return fmt.Errorf("error writting error type, %w", err)
	}
	if err := c.writer.Write(msg); err != nil {
		return fmt.Errorf("error writting error reply, %w", err)
	}
	return nil
}

func (c *Conn) WriteString(value string) error {
	if err := c.writer.Write(resp.RespTypeString); err != nil {
		return fmt.Errorf("error writting string type, %w", err)
	}
	if err := c.WriteLength(len(value)); err != nil {
		return fmt.Errorf("error writting string length, %w", err)
	}
	if err := c.writer.Write(value); err != nil {
		return fmt.Errorf("error writting string reply, %w", err)
	}
	return nil
}

func (c *Conn) WriteInteger(value int) error {
	if err := c.writer.Write(resp.RespTypeInt); err != nil {
		return fmt.Errorf("error writting int type, %w", err)
	}
	if err := c.writer.Write(value); err != nil {
		return fmt.Errorf("error writting int reply, %w", err)
	}
	return nil
}

func (c *Conn) WriteArray(values []string) error {
	if err := c.writer.Write(resp.RespTypeArray); err != nil {
		return fmt.Errorf("error writting array type, %w", err)
	}
	if err := c.writer.Write(len(values)); err != nil {
		return fmt.Errorf("error writting array length, %w", err)
	}
	for _, val := range values {
		if err := c.WriteString(val); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) WriteNull(respType byte) error {
	if err := c.writer.Write(respType); err != nil {
		return fmt.Errorf("error writting array type, %w", err)
	}
	if err := c.writer.Write(commands.NullString); err != nil {
		return fmt.Errorf("error writting string reply, %w", err)
	}
	return nil
}

func (c *Conn) WriteLength(length int) error {
	if err := c.writer.Write(length); err != nil {
		return err
	}
	return nil
}
