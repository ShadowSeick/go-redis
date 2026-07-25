package main

import (
	"bytes"
	"fmt"
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

	if err := c.processCommands(cmds); err != nil {
		logger.Error("error processing commands", err)
	}
	c.writer.Flush()
}

func (c *Conn) processCommands(cmds []commands.Command) error {
	for _, cmd := range cmds {
		if err := c.processCommand(cmd); err != nil {
			return fmt.Errorf("error processing command, %s: %w", cmd.String(), err)
		}
	}

	return nil
}

func (c *Conn) processCommand(command commands.Command) error {
	switch cmd := command.(type) {
	case commands.Ping:
		return c.writeStatus([]byte("PONG"))
	case *commands.Echo:
		if err := cmd.Error(); err != nil {
			return err
		}
		return c.writeString([]byte(cmd.Val))
	case *commands.Set:
		if err := cmd.Error(); err != nil {
			return err
		}

		memory.Set(cmd.Key, cmd.Value, cmd.Expiry)
		return c.writeStatus([]byte("OK"))
	case *commands.Get:
		if err := cmd.Error(); err != nil {
			return err
		}

		res := []byte(commands.NullString)
		val := memory.Get(cmd.Key)
		if val != nil {
			switch v := (*val).(type) {
			case string:
				res = []byte(v)
			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		return c.writeString(res)
	case *commands.RPush:
		if err := cmd.Error(); err != nil {
			return err
		}

		var res int
		list := memory.Get(cmd.Key)
		if list == nil {
			memory.Set(cmd.Key, cmd.Elements, 0)
		} else {
			switch t := (*list).(type) {
			case []string:
				res += len(t)
				*list = append(t, cmd.Elements...)
			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		res += len(cmd.Elements)
		return c.writeInteger([]byte(strconv.Itoa(res)))
	case *commands.LPush:
		if err := cmd.Error(); err != nil {
			return err
		}

		var res int
		list := memory.Get(cmd.Key)
		if list == nil {
			memory.Set(cmd.Key, cmd.Elements, 0)
		} else {
			switch t := (*list).(type) {
			case []string:
				res += len(t)
				*list = append(cmd.Elements, t...)
			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		res += len(cmd.Elements)
		return c.writeInteger([]byte(strconv.Itoa(res)))
	case *commands.LRange:
		if err := cmd.Error(); err != nil {
			return err
		}

		list := memory.Get(cmd.Key)
		var res []string
		if list != nil {
			switch t := (*list).(type) {
			case []string:
				start := cmd.Start
				stop := cmd.Stop
				length := len(t)

				if start < 0 {
					start = max(length+start, 0)
				}

				if cmd.Stop < 0 {
					stop = max(length+stop, 0)
				}

				if start < stop && start < len(t) {
					if stop >= len(t) {
						res = t[start:]
					} else {
						res = t[start : stop+1]
					}
				}

			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		return c.writeArray(res)
	case *commands.LLen:
		if err := cmd.Error(); err != nil {
			return err
		}

		list := memory.Get(cmd.Key)
		var res int
		if list != nil {
			switch t := (*list).(type) {
			case []string:
				res = len(t)
			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		return c.writeInteger([]byte(strconv.Itoa(res)))
	case *commands.LPop:
		if err := cmd.Error(); err != nil {
			return err
		}

		list := memory.Get(cmd.Key)
		res := []byte(commands.NullString)
		if list != nil {
			switch t := (*list).(type) {
			case []string:
				res, *list = []byte(t[0]), t[1:]
			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		return c.writeString(res)
	default:
		return fmt.Errorf("command not implemented")
	}
}

func (c *Conn) writeStatus(status []byte) error {
	if err := c.writer.WriteType(resp.RespTypeStatus); err != nil {
		return fmt.Errorf("error writting status type")
	}
	if err := c.writer.WriteReply(status); err != nil {
		return fmt.Errorf("error writting status reply, %w", err)
	}
	return nil
}

func (c *Conn) writeString(value []byte) error {
	if err := c.writer.WriteType(resp.RespTypeString); err != nil {
		return fmt.Errorf("error writting string type, %w", err)
	}
	if !bytes.Equal(value, []byte(commands.NullString)) {
		if err := c.writeLength(len(value)); err != nil {
			return fmt.Errorf("error writting string length, %w", err)
		}
	}
	if err := c.writer.WriteReply(value); err != nil {
		return fmt.Errorf("error writting string reply, %w", err)
	}
	return nil
}

func (c *Conn) writeInteger(value []byte) error {
	if err := c.writer.WriteType(resp.RespTypeInt); err != nil {
		return fmt.Errorf("error writting int type, %w", err)
	}
	if err := c.writer.WriteReply([]byte(value)); err != nil {
		return fmt.Errorf("error writting int reply, %w", err)
	}
	return nil
}

func (c *Conn) writeArray(values []string) error {
	if err := c.writer.WriteType(resp.RespTypeArray); err != nil {
		return fmt.Errorf("error writting array type, %w", err)
	}
	if err := c.writeLength(len(values)); err != nil {
		return fmt.Errorf("error writting array length, %w", err)
	}
	for _, val := range values {
		if err := c.writeString([]byte(val)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) writeLength(length int) error {
	val := strconv.Itoa(length)
	if err := c.writer.WriteReply([]byte(val)); err != nil {
		return err
	}
	return nil
}
