package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/logging"
)

const defaultCleanTime = 100 * time.Millisecond

var (
	// Flags
	port int

	logger           logging.Logger
	memory           datastructures.LRU
	blockedConnQueue = newBlpopQueue()
	unblockConnQueue = datastructures.NewFifoQueue[UnblockConn]()
)

func main() {
	config := net.ListenConfig{}
	ctx := context.Background()
	logger = logging.NewStructuredLogger()
	memory = datastructures.NewExpiryLRU()

	flag.IntVar(&port, "port", 6379, "port where redis server TCP connection will bind")
	flag.Parse()

	l, err := config.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		logger.Error("failed to bind to port 6379", err)
		os.Exit(1)
	}
	logger.Info("TCP connection opened in port 6379")

	channel := make(chan *Conn)
	var wg sync.WaitGroup
	go listenNewConnections(ctx, l, channel, &wg)

	cleanBlockedConn := time.NewTicker(defaultCleanTime)
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

			if err := conn.Process(); err != nil {
				logger.Error("error processing connection", err)
			}

			if err := blockedConnQueue.ProcessBlockedConn(); err != nil {
				logger.Error("error processing blocked connections", err)
			}

		case <-cleanBlockedConn.C:
			blockedConnQueue.clean()

			nextTick := blockedConnQueue.nextExpiry()
			if nextTick > 0 {
				cleanBlockedConn.Reset(nextTick)
			} else {
				cleanBlockedConn.Reset(defaultCleanTime)
			}

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
					conn := NewConnection(c, logger, memory)
					m, err := conn.Peek(1)
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
