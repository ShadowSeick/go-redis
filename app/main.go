package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/logging"
)

const defaultCleanTime = 100 * time.Millisecond

var (
	// Flags
	port = flag.Int("port", 6379, "port where TCP connection will bind")

	logger           logging.Logger
	memory           datastructures.LRU
	blockedConnQueue = newBlpopQueue()
	unblockConnQueue = datastructures.NewFifoQueue[UnblockConn]()
)

func main() {
	config := net.ListenConfig{}
	ctx := context.Background()
	logger = logging.NewStructuredLogger()

	flag.Parse()

	// Profile
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	l, err := config.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		logger.Error("failed to bind to port 6379", err)
		os.Exit(1)
	}
	logger.Info("TCP connection opened in port 6379")

	memory = datastructures.NewExpiryLRU()
	client := Client{memory: memory}
	cleanBlockedConn := time.NewTicker(defaultCleanTime)
	for {
		c, err := l.Accept()
		if err != nil {
			logger.Error("error accepting connection", err)
			os.Exit(1)
		}
		logger.Info("accepted connection from", "connection", c.RemoteAddr())

		for {
			select {
			case <-ctx.Done():
				c.Close()
				return
			case <-cleanBlockedConn.C:
				blocked, err := blockedConnQueue.clean()
				if err != nil {
					logger.Error("error cleaning blocked connections", err)
					continue
				}

				if err = client.processBlockConn(blocked); err != nil {
					logger.Error("error processing blocked connections", err)
				}

				nextTick := blockedConnQueue.nextExpiry()
				if nextTick > 0 {
					cleanBlockedConn.Reset(nextTick)
				} else {
					cleanBlockedConn.Reset(defaultCleanTime)
				}
			default:
				client.Reset(c)
				m, err := client.Peek(1)
				if err != nil {
					logger.Error("error reading data from client", err)
					client.Close()
					return
				}

				if len(m) == 0 {
					continue
				}

				if err = client.Process(); err != nil {
					logger.Error("error processing connection", err)
				}

				blocked, err := blockedConnQueue.getUnblockedConn()
				if err != nil {
					logger.Error("error getting unblocked connections", err)
				}

				if err = client.processBlockConn(blocked); err != nil {
					logger.Error("error processing blocked connections", err)
				}
			}
		}
	}
}
