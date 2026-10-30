package sstable

import "errors"

const (
	MagicNumber   uint64 = 0x4C534D5452454531 // "LSMTREE1"
	FooterSize    int64  = 48                 // 8 + 8*5
	DefaultBlockSize     = 4096               // 4 KB
)

var (
	ErrInvalidSSTable = errors.New("invalid or corrupted sstable file")
	ErrKeyNotFound    = errors.New("key not found in sstable")
)

// BlockHandle stores the file location of a data block.
type BlockHandle struct {
	Offset uint64
	Length uint64
}

// IndexEntry maps a block's first key to its file offset and length.
type IndexEntry struct {
	Key    []byte
	Handle BlockHandle
}

// Footer stores offsets and metadata written at the tail of every SSTable.
type Footer struct {
	IndexHandle BlockHandle
	BloomHandle BlockHandle
	EntryCount  uint64
	Magic       uint64
}

// Entry is a single key-value record with tombstone indicator.
type Entry struct {
	Key         []byte
	Value       []byte
	IsTombstone bool
}
