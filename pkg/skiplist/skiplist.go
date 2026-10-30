package skiplist

import (
	"bytes"
	"math/rand"
	"sync"
	"time"
)

const (
	maxLevel    = 16
	probability = 0.5
)

// Node represents an element in the SkipList.
type Node struct {
	key         []byte
	value       []byte
	isTombstone bool
	forward     []*Node
}

// SkipList is a thread-safe ordered key-value map.
type SkipList struct {
	mu       sync.RWMutex
	header   *Node
	level    int
	length   int
	byteSize int64
	rnd      *rand.Rand
}

// New creates a new SkipList.
func New() *SkipList {
	return &SkipList{
		header: &Node{forward: make([]*Node, maxLevel)},
		level:  1,
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SkipList) randomLevel() int {
	lvl := 1
	for lvl < maxLevel && s.rnd.Float64() < probability {
		lvl++
	}
	return lvl
}

// Put inserts or updates a key-value pair.
func (s *SkipList) Put(key, value []byte, isTombstone bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	update := make([]*Node, maxLevel)
	curr := s.header

	for i := s.level - 1; i >= 0; i-- {
		for curr.forward[i] != nil && bytes.Compare(curr.forward[i].key, key) < 0 {
			curr = curr.forward[i]
		}
		update[i] = curr
	}

	curr = curr.forward[0]
	if curr != nil && bytes.Equal(curr.key, key) {
		// Key exists: update in-place
		s.byteSize += int64(len(value) - len(curr.value))
		curr.value = bytes.Clone(value)
		curr.isTombstone = isTombstone
		return
	}

	lvl := s.randomLevel()
	if lvl > s.level {
		for i := s.level; i < lvl; i++ {
			update[i] = s.header
		}
		s.level = lvl
	}

	newNode := &Node{
		key:         bytes.Clone(key),
		value:       bytes.Clone(value),
		isTombstone: isTombstone,
		forward:     make([]*Node, lvl),
	}

	for i := 0; i < lvl; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	s.length++
	s.byteSize += int64(len(key) + len(value) + 16)
}

// Get searches for a key. Returns (value, isTombstone, found).
func (s *SkipList) Get(key []byte) ([]byte, bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	curr := s.header
	for i := s.level - 1; i >= 0; i-- {
		for curr.forward[i] != nil && bytes.Compare(curr.forward[i].key, key) < 0 {
			curr = curr.forward[i]
		}
	}

	curr = curr.forward[0]
	if curr != nil && bytes.Equal(curr.key, key) {
		return bytes.Clone(curr.value), curr.isTombstone, true
	}
	return nil, false, false
}

// ByteSize returns approximate memory consumed by keys and values.
func (s *SkipList) ByteSize() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byteSize
}

// Len returns the number of nodes.
func (s *SkipList) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.length
}

// Iterator represents an ordered iterator over the SkipList snapshot.
type Iterator struct {
	curr *Node
}

// NewIterator creates an iterator positioned before the first element.
func (s *SkipList) NewIterator() *Iterator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &Iterator{curr: s.header}
}

// Next advances the iterator.
func (it *Iterator) Next() bool {
	if it.curr == nil {
		return false
	}
	it.curr = it.curr.forward[0]
	return it.curr != nil
}

// Valid checks if current position is valid.
func (it *Iterator) Valid() bool {
	return it.curr != nil && it.curr.key != nil
}

// Key returns the current key.
func (it *Iterator) Key() []byte {
	return it.curr.key
}

// Value returns the current value.
func (it *Iterator) Value() []byte {
	return it.curr.value
}

// IsTombstone returns whether current entry is deleted.
func (it *Iterator) IsTombstone() bool {
	return it.curr.isTombstone
}
