package main

import (
	"bytes"
	"net"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

type Conn struct {
	conn   net.Conn
	reader *resp.Reader
	writer *resp.Writer
}

func NewConnection(conn net.Conn) *Conn {
	return &Conn{
		conn:   conn,
		reader: resp.NewReader(conn),
		writer: resp.NewWriter(conn),
	}
}

func (c *Conn) Close() {
	c.conn.Close()
}

func (c *Conn) Process() {
	reply, err := c.reader.ReadReply()
	if err != nil {
		logger.Error("error reading client reply", err)
		return
	}

	cmds := make([]commands.Command, 0)
	switch v := reply.(type) {
	case string: // simple command
		cmd := strings.ToLower(v)
		command, err := commands.New(cmd)
		if err != nil {
			logger.Error("error parsing string", err)
			return
		}
		cmds = append(cmds, command)
	case []any: // Multiple command.
		vals := make([]string, len(v))
		for i, s := range v {
			switch t := s.(type) {
			case string:
				vals[i] = t
			default:
				panic("type not expected")
			}
		}

		cmd := strings.ToLower(vals[0])
		command, err := commands.New(cmd)
		if err != nil {
			logger.Error("error parsing array string", err)
			return
		}
		command.SetArgs(vals[1:])
		cmds = append(cmds, command)
	default:
		logger.Info("type not expected", v)
		panic("type not expected")
	}

	c.processCommands(cmds)
	c.writer.Flush()
}

func (c *Conn) processCommands(cmds []commands.Command) {
	for _, cmd := range cmds {
		c.processCommand(cmd)
	}
}

func (c *Conn) processCommand(command commands.Command) {
	switch cmd := command.(type) {
	case commands.Ping:
		c.writeStatus(cmd)
	case *commands.Echo:
		if err := cmd.Error(); err != nil {
			logger.Error("error getting result from ECHO", err)
			return
		}
		c.writeString(cmd)
	case *commands.Set:
		if err := cmd.Process(memory); err != nil {
			logger.Error("error processing set", err)
			return
		}
		c.writeStatus(cmd)
	case *commands.Get:
		if err := cmd.Process(memory); err != nil {
			logger.Error("error processing get", err)
			return
		}
		c.writeString(cmd)
	case *commands.RPush:
		if err := cmd.Process(memory); err != nil {
			logger.Error("error processing rpush", err)
			return
		}
		res := cmd.Result()
		if err := c.writer.WriteType(resp.RespTypeInt); err != nil {
			logger.Error("error writing to connection", err)
			return
		}
		if err := c.writer.WriteReply([]byte(res)); err != nil {
			logger.Error("error writing to connection", err)
			return
		}
	default:
		logger.Warn("command not implemented", cmd.String())
	}
}

func (c *Conn) writeStatus(cmd commands.Command) {
	c.writer.WriteType(resp.RespTypeStatus)
	c.writer.WriteReply(cmd.Result())
}

func (c *Conn) writeString(cmd commands.Command) {
	res := cmd.Result()
	if err := c.writer.WriteType(resp.RespTypeString); err != nil {
		logger.Error("error writing to connection", err)
	}

	if !bytes.Equal(res, []byte(commands.NullString)) {
		val := strconv.Itoa(len(res))
		if err := c.writer.WriteReply([]byte(val)); err != nil {
			logger.Error("error writing to connection 2", err)
		}
	}

	if err := c.writer.WriteReply(res); err != nil {
		logger.Error("error writing to connection 3", err)
	}
}
