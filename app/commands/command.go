package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	NullString = "-1"

	// Errors
	ErrNotValidCommand      = errors.New("not a valid command")
	ErrTypeNotAllowed       = errors.New("not a valid type")
	ErrNotValidArgType      = errors.New("not a valid argument type")
	ErrNotValidNumberOfArgs = errors.New("not a valid number of args")
)

type CommandType uint8

const (
	CommandTypePing CommandType = iota
	CommandTypeEcho
	CommandTypeSet
	CommandTypeGet
	CommandTypeRPush
	CommandTypeLRange
	CommandTypeCount
)

var commandTypeStrings = [CommandTypeCount]string{
	CommandTypePing:   "ping",
	CommandTypeEcho:   "echo",
	CommandTypeSet:    "set",
	CommandTypeGet:    "get",
	CommandTypeRPush:  "rpush",
	CommandTypeLRange: "lrange",
}

func (ct CommandType) String() string {
	return commandTypeStrings[ct]
}

func New(name string) (Command, error) {
	for i, v := range commandTypeStrings {
		if v == name {
			switch CommandType(i) {
			case CommandTypePing:
				return Ping{}, nil
			case CommandTypeEcho:
				return &Echo{}, nil
			case CommandTypeSet:
				return &Set{}, nil
			case CommandTypeGet:
				return &Get{}, nil
			case CommandTypeRPush:
				return &RPush{}, nil
			case CommandTypeLRange:
				return &LRange{}, nil
			}
		}
	}
	return nil, ErrNotValidCommand
}

type Command interface {
	String() string
	SetArgs(...any)
	Error() error
}

type Ping struct{}

func (p Ping) String() string {
	return CommandTypePing.String()
}

func (p Ping) SetArgs(_ ...any) {}

func (p Ping) Error() error {
	return nil
}

type Echo struct {
	Val string
	err error
}

func (e *Echo) String() string {
	return CommandTypeEcho.String()
}

func (e *Echo) SetArgs(args ...any) {
	for _, v := range args {
		switch t := v.(type) {
		case []string:
			e.Val = t[0]
		default:
			e.err = ErrTypeNotAllowed
		}
	}
}

func (e *Echo) Error() error {
	return e.err
}

type Set struct {
	Key    string
	Value  string
	Expiry int
	err    error
}

var (
	ErrNotValidExpiryValue = errors.New("not a valid expirty value")
	ErrNotValidSetOption   = errors.New("not a valid SET option")
)

// Set Options
const (
	PX = "px"
	EX = "ex"
)

func (s *Set) String() string {
	return CommandTypeSet.String()
}

func (s *Set) SetArgs(args ...any) {
	for _, v := range args {
		switch t := v.(type) {
		case []string:
			if len(t) != 2 && len(t) != 4 {
				s.err = ErrNotValidNumberOfArgs
				return
			}

			s.Key = t[0]
			s.Value = t[1]
			if len(t) == 4 {
				expiry, err := strconv.Atoi(t[3])
				if err != nil {
					s.err = ErrNotValidExpiryValue
					return
				}
				switch strings.ToLower(t[2]) {
				case PX:
					s.Expiry = expiry
				case EX:
					s.Expiry = expiry * 1_000
				default:
					s.err = ErrNotValidSetOption
					return
				}
			}
		default:
			s.err = ErrTypeNotAllowed
			return
		}
	}
}

func (s *Set) Error() error {
	return s.err
}

type Get struct {
	Key string
	err error
}

func (g *Get) String() string {
	return CommandTypeGet.String()
}

func (g *Get) SetArgs(args ...any) {
	for _, v := range args {
		switch t := v.(type) {
		case []string:
			g.Key = t[0]
		default:
			g.err = ErrTypeNotAllowed
		}
	}
}

func (g *Get) Error() error {
	return g.err
}

type RPush struct {
	Key      string
	Elements []string
	res      int
	err      error
}

func (rp *RPush) String() string {
	return CommandTypeRPush.String()
}

func (rp *RPush) SetArgs(args ...any) {
	for _, v := range args {
		switch t := v.(type) {
		case []string:
			if len(t) < 2 {
				rp.err = ErrNotValidNumberOfArgs
				return
			}
			rp.Key = t[0]
			rp.Elements = t[1:]
		default:
			rp.err = ErrTypeNotAllowed
		}
	}
}

func (rp *RPush) Error() error {
	return rp.err
}

type LRange struct {
	Key   string
	Start int
	Stop  int
	err   error
}

func (lr *LRange) String() string {
	return CommandTypeLRange.String()
}

func (lr *LRange) SetArgs(args ...any) {
	for _, arg := range args {
		switch list := arg.(type) {
		case []string:
			if len(list) != 3 {
				lr.err = ErrNotValidNumberOfArgs
				return
			}

			lr.Key = list[0]

			start, err := strconv.Atoi(list[1])
			if err != nil {
				lr.err = fmt.Errorf("%w, %w", ErrNotValidArgType, err)
				return
			}

			stop, err := strconv.Atoi(list[2])
			if err != nil {
				lr.err = fmt.Errorf("%w, %w", ErrNotValidArgType, err)
				return
			}

			lr.Start = start
			lr.Stop = stop

		default:
			lr.err = ErrTypeNotAllowed
		}
	}
}

func (lr *LRange) Error() error {
	return lr.err
}
