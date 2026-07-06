package main

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/logging"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

var logger logging.Logger

type ConnectionRequest struct {
	writer  *resp.Writer
	command commands.Command
}

func main() {
	config := net.ListenConfig{}
	ctx := context.Background()
	logger = logging.NewStructuredLogger()

	l, err := config.Listen(ctx, "tcp", "0.0.0.0:6379")
	if err != nil {
		logger.Error("failed to bind to port 6379", err)
		os.Exit(1)
	}
	logger.Info("TCP connection opened in port 6379")

	clientChannel := make(chan ConnectionRequest)
	var wg sync.WaitGroup
	go listenNewConnections(ctx, l, clientChannel, &wg)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			close(clientChannel)
			return
		case data, ok := <-clientChannel:
			if !ok {
				return
			}

			switch data.command.(type) {
			case commands.Ping:
				res, err := data.command.Result()
				if err != nil {
					logger.Error("error getting result from PING", err)
					return
				}
				data.writer.WriteType(resp.RespTypeStatus)
				data.writer.WriteReply(res)
			case *commands.Echo:
				res, err := data.command.Result()
				if err != nil {
					logger.Error("error getting result from ECHO", err)
					return
				}

				if err := data.writer.WriteType(resp.RespTypeString); err != nil {
					logger.Error("error writing to connection", err)
				}
				val := strconv.Itoa(len(res))
				if err := data.writer.WriteReply([]byte(val)); err != nil {
					logger.Error("error writing to connection 2", err)
				}
				if err := data.writer.WriteReply(res); err != nil {
					logger.Error("error writing to connection 3", err)
				}

			default:
				logger.Warn("command not implemented", data.command.String())
			}

			data.writer.Flush()
		}
	}
}

func listenNewConnections(ctx context.Context, l net.Listener, clientChannel chan ConnectionRequest, wg *sync.WaitGroup) {
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Error("error accepting connection", err)
			os.Exit(1)
		}
		logger.Info("accepted connection from", "connection", conn.RemoteAddr())

		// Spawn new goroutine each time a new connection is accepted
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					conn.Close()
					return
				default:
					reader := resp.NewReader(conn)
					m, err := reader.Peek(1)
					if err != nil {
						logger.Error("error reading data from client", err)
						conn.Close()
						return
					}

					if len(m) == 0 {
						continue
					}

					connReq := ConnectionRequest{
						writer: resp.NewWriter(conn),
					}

					reply, err := reader.ReadReply()
					if err != nil {
						logger.Error("error reading client reply", err)
						return
					}

					switch v := reply.(type) {
					case string: // simple command
						cmd := strings.ToLower(v)
						command, err := commands.New(cmd)
						if err != nil {
							logger.Error("error parsing string", err)
							return
						}
						connReq.command = command
					case []any: // Multiple command. I need a better way of doing this
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
						connReq.command = command
					default:
						logger.Info("type not expected", v)
						panic("type not expected")
					}

					clientChannel <- connReq
				}
			}
		})
	}
}
