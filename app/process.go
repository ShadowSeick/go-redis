package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func (c *Conn) Process() error {
	reply, err := c.ReadReply()
	if err != nil {
		return fmt.Errorf("error reading client reply: %w", err)
	}

	cmds := make([]commands.Command, 0)
	switch v := reply.(type) {
	case string: // simple command
		cmd := strings.ToLower(v)
		command, err := commands.New(cmd)
		if err != nil {
			return fmt.Errorf("error parsing string: %w", err)
		}
		cmds = append(cmds, command)
	case []any: // Multiple command.
		vals := make([]string, len(v))
		for i, s := range v {
			switch t := s.(type) {
			case string:
				vals[i] = t
			default:
				return commands.ErrTypeNotAllowed
			}
		}

		cmd := strings.ToLower(vals[0])
		command, err := commands.New(cmd)
		if err != nil {
			return fmt.Errorf("error parsing array string: %w", err)
		}
		command.SetArgs(vals[1:])
		cmds = append(cmds, command)
	default:
		return commands.ErrTypeNotAllowed
	}

	if err := c.processCommands(cmds); err != nil {
		return fmt.Errorf("error processing commands: %w", err)
	}
	return c.Flush()
}

func (c *Conn) processCommands(cmds []commands.Command) error {
	for _, cmd := range cmds {
		if err := c.ProcessCommand(cmd); err != nil {
			return fmt.Errorf("error processing command, %s: %w", cmd.String(), err)
		}
	}

	return nil
}

func (c *Conn) ProcessCommand(command commands.Command) error {
	switch cmd := command.(type) {
	case commands.Ping:
		return c.WriteStatus([]byte("PONG"))
	case *commands.Echo:
		if err := cmd.Error(); err != nil {
			return err
		}
		return c.WriteString([]byte(cmd.Val))
	case *commands.Set:
		if err := cmd.Error(); err != nil {
			return err
		}

		memory.Set(cmd.Key, cmd.Value, cmd.Expiry)
		return c.WriteStatus([]byte("OK"))
	case *commands.Get:
		if err := cmd.Error(); err != nil {
			return err
		}

		var res []byte
		val := memory.Get(cmd.Key)
		if val != nil {
			switch v := (*val).(type) {
			case string:
				res = []byte(v)
			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		if len(res) == 0 {
			return c.WriteNull(resp.RespTypeString)
		} else {
			return c.WriteString(res)
		}
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

		unblockConnQueue.Push(UnblockConn{Key: cmd.Key, events: len(cmd.Elements)})
		res += len(cmd.Elements)
		return c.WriteInteger([]byte(strconv.Itoa(res)))
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

		unblockConnQueue.Push(UnblockConn{Key: cmd.Key, events: len(cmd.Elements)})
		res += len(cmd.Elements)
		return c.WriteInteger([]byte(strconv.Itoa(res)))
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

		return c.WriteArray(res)
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

		return c.WriteInteger([]byte(strconv.Itoa(res)))
	case *commands.LPop:
		if err := cmd.Error(); err != nil {
			return err
		}

		list := memory.Get(cmd.Key)
		var res []string
		if list != nil {
			switch t := (*list).(type) {
			case []string:
				if len(t) > 0 {
					index := min(cmd.Count, len(t))
					index = max(index, 1)
					if index >= len(t) {
						res = t[:]
						*list = nil
					} else {
						res, *list = t[0:index], t[index:]
					}
				}
			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		if len(res) == 0 {
			res = append(res, commands.NullString)
		}

		if len(res) == 0 {
			return c.WriteNull(resp.RespTypeString)
		} else if len(res) == 1 {
			return c.WriteString([]byte(res[0]))
		} else {
			return c.WriteArray(res)
		}

	case *commands.BLPop:
		if err := cmd.Error(); err != nil {
			return err
		}

		if cmd.Unblock {
			list := memory.Get(cmd.Keys[0])
			res := []string{cmd.Keys[0]}
			if list != nil {
				switch t := (*list).(type) {
				case []string:
					if len(t) > 0 {
						res, *list = append(res, t[0]), t[1:]
						fmt.Println(res)
					}
				default:
					return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
				}
			}

			if len(res) > 1 {
				return c.WriteArray(res)
			} else {
				return c.WriteNull(resp.RespTypeArray)
			}
		}

		blockedConnQueue.Push(blockConn{keys: cmd.Keys, expiry: int(cmd.Expiry), conn: c})
		return nil
	default:
		return fmt.Errorf("command not implemented")
	}
}
