package main

import (
	"log/slog"
	"net"
	"os"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		slog.Error("failed to bind to port 6379", "err", err)
		os.Exit(1)
	}
	slog.Info("TCP connection opened in port 6379")

	conn, err := l.Accept()
	if err != nil {
		slog.Error("error accepting connection", "err", err)
		os.Exit(1)
	}

	defer conn.Close()
	incomingData := make([]byte, 256)
	for {
		n, err := conn.Read(incomingData)
		if err != nil {
			slog.Error("error reading data from connection", "err", err)
			return
		}

		if n == 0 {
			continue
		}

		_, err = conn.Write([]byte("+PONG\r\n"))
		if err != nil {
			slog.Error("error writing data to connection", "err", err)
			return
		}
	}
}
