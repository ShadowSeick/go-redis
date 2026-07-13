package commands

import (
	"errors"
	"fmt"
	"strings"
)

var (
	nullString = "-1"

	// Errors
	ErrNotValidCommand = errors.New("not a valid command")
	ErrTypeNotAllowed  = errors.New("not a valid type")
)

type CommandType uint8

const (
	CommandTypePing CommandType = iota
	CommandTypeEcho
	CommandTypeSet
	CommandTypeGet
	CommandTypeCount
)

var commandTypeStrings = [CommandTypeCount]string{
	CommandTypePing: "ping",
	CommandTypeEcho: "echo",
	CommandTypeSet:  "set",
	CommandTypeGet:  "get",
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
			}
		}
	}
	return nil, ErrNotValidCommand
}

type Command interface {
	String() string
	SetArgs(...any)
	Result() []byte
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

func (p Ping) Result() []byte {
	return []byte("PONG")
}

type Echo struct {
	val string
	err error
}

func (e *Echo) String() string {
	return CommandTypeEcho.String()
}

func (e *Echo) SetArgs(args ...any) {
	var builder strings.Builder
	for _, v := range args {
		switch t := v.(type) {
		case []string:
			for _, s := range t {
				builder.WriteString(s)
			}
			e.val = builder.String()
		default:
			e.err = ErrTypeNotAllowed
		}
	}
}

func (e *Echo) Error() error {
	return e.err
}

func (e *Echo) Result() []byte {
	return []byte(e.val)
}

type Set struct {
	val []string
	err error
}

func (s *Set) String() string {
	return CommandTypeSet.String()
}

func (s *Set) SetArgs(args ...any) {
	for _, v := range args {
		switch t := v.(type) {
		case []string:
			if len(t) != 2 {
				s.err = fmt.Errorf("not a valid number of args %d", len(t))
				return
			}

			s.val = t
		default:
			s.err = ErrTypeNotAllowed
		}
	}
}

func (s *Set) Error() error {
	return s.err
}

func (s *Set) Process(hashMap map[string]any) error {
	if s.err != nil {
		return s.err
	}

	hashMap[s.val[0]] = s.val[1]
	return nil
}

func (s *Set) Result() []byte {
	return []byte("OK")
}

type Get struct {
	val any
	err error
}

func (g *Get) String() string {
	return CommandTypeGet.String()
}

func (g *Get) SetArgs(args ...any) {
	var builder strings.Builder
	for _, v := range args {
		switch t := v.(type) {
		case []string:
			for _, s := range t {
				builder.WriteString(s)
			}
			g.val = builder.String()
		default:
			g.err = ErrTypeNotAllowed
		}
	}
}

func (g *Get) Error() error {
	return g.err
}

func (g *Get) Process(hashMap map[string]any) error {
	if g.err != nil {
		return g.err
	}

	switch v := g.val.(type) {
	case string:
		val, ok := hashMap[v]
		if !ok {
			g.val = nullString
			return nil
		}

		g.val = val
	default:
		g.err = ErrTypeNotAllowed
	}

	return g.err
}

func (g *Get) Result() []byte {
	switch v := g.val.(type) {
	case string:
		return []byte(v)
	default:
		panic(ErrTypeNotAllowed.Error())
	}
}
