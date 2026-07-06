package commands

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var ErrNotValidCommand = errors.New("not a valid command")

type CommandType uint8

const (
	CommandTypePing CommandType = iota
	CommandTypeEcho
	CommandTypeCount
)

var commandTypeStrings = [CommandTypeCount]string{
	CommandTypePing: "ping",
	CommandTypeEcho: "echo",
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
			}
		}
	}
	return nil, ErrNotValidCommand
}

type Command interface {
	String() string
	SetArgs(...any)
	Result() ([]byte, error)
}

type Ping struct{}

func (p Ping) String() string {
	return CommandTypePing.String()
}
func (p Ping) SetArgs(_ ...any) {}
func (p Ping) Result() ([]byte, error) {
	return []byte("PONG"), nil
}

type Echo struct {
	val string
}

func (e *Echo) String() string {
	return CommandTypeEcho.String()
}

func (e *Echo) SetArgs(args ...any) {
	var builder strings.Builder
	fmt.Println(reflect.TypeOf(args))
	for _, v := range args {
		switch t := v.(type) {
		case string:
			builder.WriteString(t)
		case byte:
			builder.WriteByte(t)
		case rune:
			builder.WriteRune(t)
		default:
			fmt.Println(reflect.TypeOf(t))
			panic("no other type is accepted")
		}
	}
	e.val = builder.String()
}

func (e *Echo) Result() ([]byte, error) {
	return []byte(e.val), nil
}
