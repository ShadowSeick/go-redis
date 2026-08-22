package main

import (
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
)

type blockConn struct {
	keys   []string
	expiry int
	conn   net.Conn
}

func (bc blockConn) ID() string {
	var builder strings.Builder
	for _, key := range bc.keys {
		builder.WriteString(key)
	}
	builder.WriteString(strconv.Itoa(bc.expiry))
	// This should be unique for the connection.
	// For now it will work, but maybe it's a good idea to add a unique ID
	builder.WriteString(bc.conn.LocalAddr().String())
	builder.WriteString(bc.conn.RemoteAddr().String())
	return builder.String()
}

func (bc blockConn) Equals(block blockConn) bool {
	return bc.ID() == block.ID()
}

type UnblockConn struct {
	Key    string
	events int
}

func (uc UnblockConn) Equals(unblock UnblockConn) bool {
	return uc.Key == unblock.Key
}

type blpopQueue struct {
	keysEvents map[string]datastructures.Queue[blockConn]
	events     datastructures.LRU
	nextClean  []time.Time
}

func newBlpopQueue() *blpopQueue {
	return &blpopQueue{
		keysEvents: make(map[string]datastructures.Queue[blockConn], 1024),
		events:     datastructures.NewExpiryLRU(),
	}
}

func (q *blpopQueue) process(unblock UnblockConn) ([]blockConn, error) {
	var blockConns []blockConn

	queue, ok := q.keysEvents[unblock.Key]
	if !ok {
		return nil, fmt.Errorf("no events for key: %s", unblock.Key)
	}

	for i := 0; i < unblock.events; i++ {
		block, err := queue.Pop()
		if err != nil {
			return nil, err
		}

		blockConns = append(blockConns, blockConn{
			keys: []string{unblock.Key},
			conn: block.conn,
		})

		if err := q.cleanBlocked(block); err != nil {
			return nil, err
		}
	}

	return blockConns, nil
}

func (q *blpopQueue) Push(block blockConn) {
	for _, key := range block.keys {
		queue, ok := q.keysEvents[key]
		if !ok {
			queue = datastructures.NewFifoQueue[blockConn]()
		}
		queue.Push(block)
		q.keysEvents[key] = queue
	}

	q.events.Set(block.ID(), block, block.expiry)
	if block.expiry > 0 {
		q.nextClean = append(q.nextClean, time.Now().Add(time.Duration(block.expiry)*time.Millisecond))
		// Sort Desc to pop constant time
		slices.SortStableFunc(q.nextClean, func(a time.Time, b time.Time) int {
			if a.Equal(b) {
				return 0
			} else if a.After(b) {
				return -1
			} else {
				return 1
			}
		})
	}
}

func (q *blpopQueue) clean() ([]blockConn, error) {
	blpopEvents := q.events.Clean()
	if len(blpopEvents) == 0 {
		return nil, nil
	}

	var blocked []blockConn
	for _, node := range blpopEvents {
		switch v := node.Value.(type) {
		case blockConn:
			blocked = append(blocked, v)
			if err := q.cleanBlocked(v); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("invalid blocked event")
		}
	}
	return blocked, nil
}

func (q *blpopQueue) nextExpiry() time.Duration {
	if len(q.nextClean) == 0 {
		return 0
	}

	nextExpiry := q.nextClean[len(q.nextClean)-1]
	q.nextClean = q.nextClean[:len(q.nextClean)-1]
	return time.Until(nextExpiry)
}

func (q *blpopQueue) cleanBlocked(block blockConn) error {
	for _, key := range block.keys {
		queue, ok := q.keysEvents[key]
		if !ok {
			return fmt.Errorf("key should be registered")
		}

		queue.Remove(block)
	}
	return nil
}

func (q *blpopQueue) getUnblockedConn() ([]blockConn, error) {
	var blockConns []blockConn
	for i := 0; i < unblockConnQueue.Len(); i++ {
		if q.events.Len() == 0 {
			unblockConnQueue.Clean()
			break
		}

		unblock, err := unblockConnQueue.Pop()
		if err != nil {
			return nil, fmt.Errorf("error popping queue: %w", err)
		}

		blocked, err := q.process(unblock)
		if err != nil {
			return nil, fmt.Errorf("error processing unblock connection: %w", err)
		}

		blockConns = append(blockConns, blocked...)
	}

	return blockConns, nil
}
