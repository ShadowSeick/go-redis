package commands

import (
	"errors"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
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
	key    string
	value  string
	expiry int
	err    error
}

var (
	ErrNotValidExpiryValue  = errors.New("not a valid expirty value")
	ErrNotValidNumberOfArgs = errors.New("not a valid number of args")
	ErrNotValidSetOption    = errors.New("not a valid SET option")
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

			s.key = t[0]
			s.value = t[1]
			if len(t) == 4 {
				expiry, err := strconv.Atoi(t[3])
				if err != nil {
					s.err = ErrNotValidExpiryValue
					return
				}
				switch strings.ToLower(t[2]) {
				case PX:
					s.expiry = expiry
				case EX:
					s.expiry = expiry * 1_000
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

func (s *Set) Process(lru datastructures.LRU) error {
	if s.err != nil {
		return s.err
	}

	lru.Set(s.key, s.value, s.expiry)
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

func (g *Get) Process(lru datastructures.LRU) error {
	if g.err != nil {
		return g.err
	}

	switch v := g.val.(type) {
	case string:
		val := lru.Get(v)
		g.val = nullString

		if val != nil {
			g.val = *val
		}
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
