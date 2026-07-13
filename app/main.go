package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/pkg/logging"
)

var logger logging.Logger

var globalMemory = make(map[string]any)

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

	channel := make(chan *Conn)
	var wg sync.WaitGroup
	go listenNewConnections(ctx, l, channel, &wg)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			close(channel)
			return
		case conn, ok := <-channel:
			if !ok {
				fmt.Println("is it ok")
				return
			}

			// Something is fishy when we have too many connection, probably I will need some sort of pool for connections
			logger.Info("process")
			conn.Process()
			logger.Info("proccessed")
		}
	}
}

func listenNewConnections(ctx context.Context, l net.Listener, clientChannel chan *Conn, wg *sync.WaitGroup) {
	for {
		c, err := l.Accept()
		if err != nil {
			logger.Error("error accepting connection", err)
			os.Exit(1)
		}
		logger.Info("accepted connection from", "connection", c.RemoteAddr())

		wg.Go(func() {
			for {
				select {
				// Pretty much sure this shouldn't be here. We don't want to close the connection when the context is done.
				// We want to close the connection when there is no more data received, when the client has closed the connection in their end
				case <-ctx.Done():
					c.Close()
					return
				default:
					conn := NewConnection(c)
					m, err := conn.reader.Peek(1)
					if err != nil {
						logger.Error("error reading data from client", err)
						conn.Close()
						return
					}
					fmt.Println("here")

					if len(m) == 0 {
						continue
					}

					clientChannel <- conn
				}
			}
		})
	}
}
