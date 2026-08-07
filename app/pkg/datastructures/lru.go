package datastructures

import (
	"errors"
	"time"
)

var ErrNodeNotFound = errors.New("node not found")

type LRU interface {
	Get(key string) *any
	Set(key string, value any, expirty int)
	Remove(key string) error
	Clean() []CleanedNode
	Len() int
}

type CleanedNode struct {
	Key   string
	Value any
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
		hash: make(map[string]*Node, 1024),
	}
}

func (elru *ExpiryLRU) Get(key string) *any {
	node, ok := elru.hash[key]
	if !ok {
		return nil
	}

	if node.expiry != nil && time.Now().After(*node.expiry) {
		elru.Remove(key)
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
	}

	elru.hash[key] = node
	elru.moveToTop(node)
}

func (elru *ExpiryLRU) Clean() []CleanedNode {
	now := time.Now()

	var cleanedNodes []CleanedNode
	for key, node := range elru.hash {
		if node.expiry != nil && now.After(*node.expiry) {
			cleanedNodes = append(cleanedNodes, CleanedNode{
				Key:   key,
				Value: node.value,
			})
		}
	}

	if len(cleanedNodes) > 0 {
		for _, node := range cleanedNodes {
			elru.Remove(node.Key)
		}
	}

	return cleanedNodes
}

func (elru *ExpiryLRU) Remove(key string) error {
	node, ok := elru.hash[key]
	if !ok {
		return ErrNodeNotFound
	}

	elru.detach(node)
	delete(elru.hash, key)

	return nil
}

func (elru *ExpiryLRU) Len() int {
	return len(elru.hash)
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
