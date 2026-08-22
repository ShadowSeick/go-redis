package main

import (
	"fmt"
	"net"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

type Client struct {
	conn   net.Conn
	memory datastructures.LRU
	reader *resp.Reader
	writer *resp.Writer
}

func (c *Client) Reset(conn net.Conn) {
	c.conn = conn
	if c.reader != nil {
		c.reader.Reset(conn)
	} else {
		c.reader = resp.NewReader(conn)
	}

	if c.writer != nil {
		c.writer.Reset(conn)
	} else {
		c.writer = resp.NewWriter(conn)
	}
}

func (c *Client) Peek(n int) ([]byte, error) {
	return c.reader.Peek(n)
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) ReadReply() (any, error) {
	return c.reader.ReadReply()
}

func (c *Client) Flush() error {
	return c.writer.Flush()
}

func (c *Client) WriteStatus(status string) error {
	if err := c.writer.Write(resp.RespTypeStatus); err != nil {
		return fmt.Errorf("error writting status type")
	}
	if err := c.writer.Write(status); err != nil {
		return fmt.Errorf("error writting status reply, %w", err)
	}
	return nil
}

func (c *Client) WriteError(msg string) error {
	if err := c.writer.Write(resp.RespTypeError); err != nil {
		return fmt.Errorf("error writting error type, %w", err)
	}
	if err := c.writer.Write(msg); err != nil {
		return fmt.Errorf("error writting error reply, %w", err)
	}
	return nil
}

func (c *Client) WriteString(value string) error {
	if err := c.writer.Write(resp.RespTypeString); err != nil {
		return fmt.Errorf("error writting string type, %w", err)
	}
	if err := c.writer.Write(len(value)); err != nil {
		return fmt.Errorf("error writting string length, %w", err)
	}
	if err := c.writer.Write(value); err != nil {
		return fmt.Errorf("error writting string reply, %w", err)
	}
	return nil
}

func (c *Client) WriteInteger(value int) error {
	if err := c.writer.Write(resp.RespTypeInt); err != nil {
		return fmt.Errorf("error writting int type, %w", err)
	}
	if err := c.writer.Write(value); err != nil {
		return fmt.Errorf("error writting int reply, %w", err)
	}
	return nil
}

func (c *Client) WriteArray(values []string) error {
	if err := c.writer.Write(resp.RespTypeArray); err != nil {
		return fmt.Errorf("error writting array type, %w", err)
	}
	if err := c.writer.Write(len(values)); err != nil {
		return fmt.Errorf("error writting array length, %w", err)
	}
	for _, val := range values {
		if err := c.writer.Write(val); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) WriteNull(respType byte) error {
	if err := c.writer.Write(respType); err != nil {
		return fmt.Errorf("error writting array type, %w", err)
	}
	if err := c.writer.Write(commands.NullString); err != nil {
		return fmt.Errorf("error writting string reply, %w", err)
	}
	return nil
}
