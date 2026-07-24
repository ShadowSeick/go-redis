package datastructures

import (
	"errors"
	"fmt"
	"time"
)

var ErrNodeNotFound = errors.New("node not found")

type LRU interface {
	Get(key string) *any
	Set(key string, value any, expirty int)
	Clean()
}

type ExpiryLRU struct {
	head *Node
	hash map[string]*Node
}

type Node struct {
	value  any
	expiry *time.Time
	prev   *Node
	next   *Node
}

func NewExpiryLRU() *ExpiryLRU {
	return &ExpiryLRU{
		hash: make(map[string]*Node, 1000),
	}
}

func (elru *ExpiryLRU) Get(key string) *any {
	node, ok := elru.hash[key]
	if !ok {
		return nil
	}

	if node.expiry != nil {
		fmt.Println("EXPIRY")
		fmt.Println(*node.expiry)
		fmt.Println(time.Now())
		fmt.Println(node.value)
	}

	if node.expiry != nil && time.Now().After(*node.expiry) {
		elru.remove(key)
		return nil
	}

	elru.moveToTop(node)
	return &node.value
}

func (elru *ExpiryLRU) Set(key string, value any, expiry int) {
	node, ok := elru.hash[key]
	if !ok {
		node = &Node{}
	}

	node.value = value
	if expiry > 0 {
		expiryTime := time.Now().Add(time.Duration(expiry) * time.Millisecond)
		node.expiry = &expiryTime
		fmt.Println(node.expiry)
	}

	elru.hash[key] = node
	fmt.Println(elru.hash)
	elru.moveToTop(node)
}

func (elru *ExpiryLRU) Clean() {
	now := time.Now()

	var expiredNodes []string
	for key, node := range elru.hash {
		if node.expiry.After(now) {
			expiredNodes = append(expiredNodes, key)
		}
	}

	if len(expiredNodes) > 0 {
		for _, key := range expiredNodes {
			elru.remove(key)
		}
	}
}

func (elru *ExpiryLRU) remove(key string) error {
	node, ok := elru.hash[key]
	if !ok {
		return ErrNodeNotFound
	}

	elru.detach(node)
	delete(elru.hash, key)

	return nil
}

func (elru *ExpiryLRU) moveToTop(node *Node) {
	currHead := elru.head
	if currHead != nil {
		currHead.prev = node
		node.next = currHead
	}
	elru.head = node
}

func (elru *ExpiryLRU) detach(node *Node) {
	prev := node.prev
	next := node.next

	if prev != nil {
		prev.next = next
	}

	if next != nil {
		next.prev = prev
	}
}
