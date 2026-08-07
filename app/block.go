package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/pkg/datastructures"
)

type blockConn struct {
	keys   []string
	expiry int
	conn   *Conn
}

func (bc blockConn) ID() string {
	var builder strings.Builder
	for _, key := range bc.keys {
		builder.WriteString(key)
	}
	builder.WriteString(strconv.Itoa(bc.expiry))
	// This should be unique for the connection.
	// For now it will work, but maybe it's a good idea to add a unique ID
	builder.WriteString(bc.conn.conn.LocalAddr().String())
	builder.WriteString(bc.conn.conn.RemoteAddr().String())
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

func (q *blpopQueue) process(unblock UnblockConn) (blockConn, error) {
	var val blockConn

	queue, ok := q.keysEvents[unblock.Key]
	if !ok {
		return val, fmt.Errorf("no events for key: %s", unblock.Key)
	}

	for i := 0; i < unblock.events; i++ {
		block, err := queue.Pop()
		if err != nil {
			return val, err
		}

		if err := block.conn.ProcessCommand(&commands.BLPop{
			Keys:    []string{unblock.Key},
			Unblock: true,
		}); err != nil {
			return val, err
		}
		block.conn.Flush()
	}

	return val, nil
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

func (q *blpopQueue) clean() {
	blpopEvents := q.events.Clean()
	if len(blpopEvents) == 0 {
		return
	}

	for _, node := range blpopEvents {
		switch v := node.Value.(type) {
		case blockConn:
			if err := v.conn.ProcessCommand(&commands.BLPop{
				Keys:    v.keys,
				Unblock: true,
			}); err != nil {
				panic(err.Error())
			}
			v.conn.Flush()

			if err := q.cleanBlocked(v); err != nil {
				panic(err.Error())
			}
		default:
			panic("invalid event")
		}
	}
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

func (q *blpopQueue) ProcessBlockedConn() error {
	for i := 0; i < unblockConnQueue.Len(); i++ {
		if q.events.Len() == 0 {
			unblockConnQueue.Clean()
			break
		}

		unblock, err := unblockConnQueue.Pop()
		if err != nil {
			return fmt.Errorf("error popping queue: %w", err)
		}

		blocked, err := q.process(unblock)
		if err != nil {
			return fmt.Errorf("error processing unblock connection: %w", err)
		}

		q.cleanBlocked(blocked)
	}

	return nil
}
