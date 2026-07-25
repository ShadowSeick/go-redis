package main

import (
	"context"
	"net"
	"os"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/logging"
)

var (
	logger logging.Logger
	memory *datastructures.ExpiryLRU
)

func main() {
	config := net.ListenConfig{}
	ctx := context.Background()
	logger = logging.NewStructuredLogger()
	memory = datastructures.NewExpiryLRU()

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
				return
			}

			conn.Process()
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

					if len(m) == 0 {
						continue
					}

					clientChannel <- conn
				}
			}
		})
	}
}
