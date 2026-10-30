package lsm

import "errors"

var (
	ErrNotFound = errors.New("key not found")
	ErrClosed   = errors.New("database is closed")
)

// Options configures the LSM database engine.
type Options struct {
	Dir                   string
	MemTableThreshold     int64 // Flush active MemTable when byte size exceeds this (e.g. 2MB)
	L0CompactionThreshold int   // Trigger L0 -> L1 compaction when L0 table count reaches this (e.g. 4)
	SyncWAL               bool  // Whether to fsync WAL on every write
	BlockSize             int   // Block size for SSTables (default 4KB)
}

// DefaultOptions returns production-ready default options for the database.
func DefaultOptions(dir string) Options {
	return Options{
		Dir:                   dir,
		MemTableThreshold:     2 * 1024 * 1024, // 2MB
		L0CompactionThreshold: 4,
		SyncWAL:               false,
		BlockSize:             4096,
	}
}
