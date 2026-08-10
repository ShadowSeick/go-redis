// loadgen replays a file of Redis commands (one per line, e.g. "SET foo bar")
// against the server over TCP, so a pprof profile can be captured under load.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	addr   = flag.String("addr", "localhost:6379", "server address")
	file   = flag.String("file", "app/scripts/commands.txt", "file with one command per line")
	rounds = flag.Int("rounds", 1, "times to replay the file")
)

func main() {
	flag.Parse()

	cmds, err := readCommands(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading commands:", err)
		os.Exit(1)
	}

	conn, err := net.Dial("tcp", *addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connecting:", err)
		os.Exit(1)
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	start := time.Now()
	sent := 0
	for range *rounds {
		for _, args := range cmds {
			writeCommand(w, args)
			if err := w.Flush(); err != nil {
				fmt.Fprintln(os.Stderr, "sending:", err)
				os.Exit(1)
			}
			if err := readReply(r); err != nil {
				fmt.Fprintln(os.Stderr, "reading reply:", err)
				os.Exit(1)
			}
			sent++
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("%d commands in %s (%.0f cmd/s)\n", sent, elapsed.Round(time.Millisecond), float64(sent)/elapsed.Seconds())
}

func readCommands(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cmds [][]string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cmds = append(cmds, strings.Fields(line))
	}
	return cmds, scanner.Err()
}

// writeCommand encodes args as a RESP array of bulk strings: *N $len arg ...
func writeCommand(w *bufio.Writer, args []string) {
	fmt.Fprintf(w, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(w, "$%d\r\n%s\r\n", len(a), a)
	}
}

// readReply consumes exactly one RESP reply, discarding its contents.
func readReply(r *bufio.Reader) error {
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}

	switch line[0] {
	case '+', ':', '_', ',', '#':
		return nil
	case '-':
		fmt.Fprintln(os.Stderr, "server error:", strings.TrimSpace(line[1:]))
		return nil
	case '$', '!', '=':
		n, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return fmt.Errorf("bad bulk length %q: %w", line, err)
		}
		if n < 0 { // null bulk string ($-1)
			return nil
		}
		_, err = io.CopyN(io.Discard, r, int64(n)+2) // payload + \r\n
		return err
	case '*':
		n, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return fmt.Errorf("bad array length %q: %w", line, err)
		}
		for range max(n, 0) { // null array (*-1) has no elements
			if err := readReply(r); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown reply type %q", line)
	}
}
