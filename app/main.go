package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
)

type ConnectionRequest struct {
	conn    net.Conn
	request []byte
}

func main() {
	config := net.ListenConfig{}
	ctx := context.Background()

	l, err := config.Listen(ctx, "tcp", "0.0.0.0:6379")
	if err != nil {
		slog.Error("failed to bind to port 6379", "err", err)
		os.Exit(1)
	}
	slog.Info("TCP connection opened in port 6379")

	clientChannel := make(chan ConnectionRequest)
	var wg sync.WaitGroup
	go listenNewConnections(ctx, l, clientChannel, &wg)

	// Naive Event loop for now
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

			fmt.Println(string(data.request))

			_, err = data.conn.Write([]byte("+PONG\r\n"))
			if err != nil {
				slog.Error("error writing data to connection", "err", err)
				return
			}
		}
	}
}

func listenNewConnections(ctx context.Context, l net.Listener, clientChannel chan ConnectionRequest, wg *sync.WaitGroup) {
	for {
		conn, err := l.Accept()
		if err != nil {
			slog.Error("error accepting connection", "err", err)
			os.Exit(1)
		}
		slog.Info("accepted connection from", "connection", conn.RemoteAddr())

		// Spawn new goroutine each time a new connection is accepted
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					conn.Close()
				default:
					clientMessage := make([]byte, 1024)
					n, err := conn.Read(clientMessage)
					if err != nil {
						slog.Error("error reading data from client", "err", err)
						conn.Close()
						return
					}

					if n == 0 {
						continue
					}

					clientChannel <- ConnectionRequest{
						conn:    conn,
						request: clientMessage,
					}
				}
			}
		})
	}
}

// How to implement an event loop
// The idea is pretty simple. We need:
// - Queue
// - Listener for the Queue
// Each queue must be independant
// Each listener must be independant
