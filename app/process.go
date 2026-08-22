package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func (c *Client) Process() error {
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
			return c.replyUnknownCommand(cmd, err)
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
			return c.replyUnknownCommand(cmd, err)
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

func (c *Client) replyUnknownCommand(cmd string, err error) error {
	if !errors.Is(err, commands.ErrNotValidCommand) {
		return fmt.Errorf("error parsing command: %w", err)
	}
	if err := c.WriteError(fmt.Sprintf("ERR unknown command '%s'", cmd)); err != nil {
		return err
	}
	return c.Flush()
}

func (c *Client) processBlockConn(blocked []blockConn) error {
	for _, block := range blocked {
		c.Reset(block.conn)
		if err := c.processCommand(&commands.LPop{Key: block.keys[0]}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) processCommands(cmds []commands.Command) error {
	for _, cmd := range cmds {
		if err := c.processCommand(cmd); err != nil {
			if wErr := c.WriteError(fmt.Sprintf("error proccessing command: %s", cmd.String())); wErr != nil {
				return wErr
			}
			return fmt.Errorf("error processing command, %s: %w", cmd.String(), err)
		}
	}

	return nil
}

func (c *Client) processCommand(command commands.Command) error {
	switch cmd := command.(type) {
	case commands.Ping:
		return c.WriteStatus("PONG")
	case *commands.Echo:
		if err := cmd.Error(); err != nil {
			return err
		}
		return c.WriteString(cmd.Val)
	case *commands.Set:
		if err := cmd.Error(); err != nil {
			return err
		}

		c.memory.Set(cmd.Key, cmd.Value, cmd.Expiry)
		return c.WriteStatus("OK")
	case *commands.Get:
		if err := cmd.Error(); err != nil {
			return err
		}

		var res string
		val := c.memory.Get(cmd.Key)
		if val != nil {
			switch v := (*val).(type) {
			case string:
				res = v
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
		list := c.memory.Get(cmd.Key)
		if list == nil {
			c.memory.Set(cmd.Key, cmd.Elements, 0)
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
		return c.WriteInteger(res)
	case *commands.LPush:
		if err := cmd.Error(); err != nil {
			return err
		}

		var res int
		list := c.memory.Get(cmd.Key)
		if list == nil {
			c.memory.Set(cmd.Key, cmd.Elements, 0)
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
		return c.WriteInteger(res)
	case *commands.LRange:
		if err := cmd.Error(); err != nil {
			return err
		}

		list := c.memory.Get(cmd.Key)
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

		list := c.memory.Get(cmd.Key)
		var res int
		if list != nil {
			switch t := (*list).(type) {
			case []string:
				res = len(t)
			default:
				return fmt.Errorf("error getting value from memory, %w", commands.ErrTypeNotAllowed)
			}
		}

		return c.WriteInteger(res)
	case *commands.LPop:
		if err := cmd.Error(); err != nil {
			return err
		}

		list := c.memory.Get(cmd.Key)
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
			return c.WriteString(res[0])
		} else {
			return c.WriteArray(res)
		}

	case *commands.BLPop:
		if err := cmd.Error(); err != nil {
			return err
		}

		if cmd.Unblock {
			list := c.memory.Get(cmd.Keys[0])
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

		blockedConnQueue.Push(blockConn{keys: cmd.Keys, expiry: int(cmd.Expiry), conn: c.conn})
		return nil
	default:
		return fmt.Errorf("command not implemented")
	}
}
