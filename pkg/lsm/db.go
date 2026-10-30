package lsm

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/sshloktiwarii/lsm-tree-db/pkg/compactor"
	"github.com/sshloktiwarii/lsm-tree-db/pkg/manifest"
	"github.com/sshloktiwarii/lsm-tree-db/pkg/skiplist"
	"github.com/sshloktiwarii/lsm-tree-db/pkg/sstable"
	"github.com/sshloktiwarii/lsm-tree-db/pkg/wal"
)

// DB represents a high-performance LSM-Tree database instance.
type DB struct {
	mu            sync.RWMutex
	dir           string
	opts          Options
	memTable      *skiplist.SkipList
	immTable      *skiplist.SkipList
	wal           *wal.WAL
	manifest      *manifest.Manifest
	tableReaders  map[uint64]*sstable.Reader
	stats         internalStats
	compactNotify chan struct{}
	closeNotify   chan struct{}
	bgWg          sync.WaitGroup
	isClosed      bool
}

// Open initializes or recovers an LSM tree database in the directory.
func Open(opts Options) (*DB, error) {
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return nil, err
	}
	m, err := manifest.Open(opts.Dir)
	if err != nil {
		return nil, err
	}

	walPath := filepath.Join(opts.Dir, "active.wal")
	replayed, err := wal.Replay(walPath)
	if err != nil {
		return nil, err
	}

	activeMem := skiplist.New()
	for _, rec := range replayed {
		activeMem.Put(rec.Key, rec.Value, rec.Type == wal.TypeDelete)
	}

	w, err := wal.Open(walPath)
	if err != nil {
		return nil, err
	}

	db := &DB{
		dir:           opts.Dir,
		opts:          opts,
		memTable:      activeMem,
		wal:           w,
		manifest:      m,
		tableReaders:  make(map[uint64]*sstable.Reader),
		compactNotify: make(chan struct{}, 1),
		closeNotify:   make(chan struct{}),
	}

	// Cache active SSTable readers
	v := m.Current()
	for _, tables := range v.Levels {
		for _, t := range tables {
			p := filepath.Join(opts.Dir, fmt.Sprintf("%06d.sst", t.FileNum))
			r, err := sstable.OpenReader(p)
			if err == nil {
				db.tableReaders[t.FileNum] = r
			}
		}
	}

	db.bgWg.Add(1)
	go db.compactionWorker()

	return db, nil
}

// Put writes or updates a key-value pair.
func (db *DB) Put(key, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.isClosed {
		return ErrClosed
	}

	if err := db.wal.Write(wal.TypePut, key, value); err != nil {
		return err
	}
	if db.opts.SyncWAL {
		_ = db.wal.Sync()
	}

	db.memTable.Put(key, value, false)
	atomic.AddUint64(&db.stats.totalPuts, 1)

	if db.memTable.ByteSize() >= db.opts.MemTableThreshold {
		if err := db.flushLocked(); err != nil {
			return err
		}
	}
	return nil
}

// Delete marks a key as deleted using a tombstone.
func (db *DB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.isClosed {
		return ErrClosed
	}

	if err := db.wal.Write(wal.TypeDelete, key, nil); err != nil {
		return err
	}
	if db.opts.SyncWAL {
		_ = db.wal.Sync()
	}

	db.memTable.Put(key, nil, true)
	atomic.AddUint64(&db.stats.totalDeletes, 1)

	if db.memTable.ByteSize() >= db.opts.MemTableThreshold {
		if err := db.flushLocked(); err != nil {
			return err
		}
	}
	return nil
}

// Flush manually forces the active MemTable to flush to an SSTable on disk.
func (db *DB) Flush() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.isClosed {
		return ErrClosed
	}
	return db.flushLocked()
}

func (db *DB) flushLocked() error {
	if db.memTable.Len() == 0 {
		return nil
	}

	fileNum := db.manifest.NextFileNumber()
	sstPath := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", fileNum))
	b, err := sstable.NewBuilder(sstPath, db.memTable.Len(), db.opts.BlockSize)
	if err != nil {
		return err
	}

	var minK, maxK []byte
	it := db.memTable.NewIterator()
	var count uint64 = 0
	for it.Next() {
		k := it.Key()
		v := it.Value()
		isTomb := it.IsTombstone()
		if len(minK) == 0 {
			minK = append([]byte(nil), k...)
		}
		maxK = append([]byte(nil), k...)
		if err := b.Add(k, v, isTomb); err != nil {
			return err
		}
		count++
	}

	if err := b.Finish(); err != nil {
		return err
	}

	stat, _ := os.Stat(sstPath)
	var sz int64
	if stat != nil {
		sz = stat.Size()
	}

	r, err := sstable.OpenReader(sstPath)
	if err != nil {
		return err
	}
	db.tableReaders[fileNum] = r

	edit := &manifest.VersionEdit{
		Added: []manifest.TableMetadata{{
			FileNum:    fileNum,
			Level:      0,
			Size:       sz,
			EntryCount: count,
			MinKey:     minK,
			MaxKey:     maxK,
		}},
	}
	if err := db.manifest.Apply(edit); err != nil {
		return err
	}

	// Rotate WAL
	_ = db.wal.Close()
	walPath := filepath.Join(db.dir, "active.wal")
	_ = os.Truncate(walPath, 0)
	newWal, err := wal.Open(walPath)
	if err != nil {
		return err
	}
	db.wal = newWal

	// Reset memtable
	db.memTable = skiplist.New()

	// Signal background compactor if L0 exceeds threshold
	v := db.manifest.Current()
	if len(v.Levels[0]) >= db.opts.L0CompactionThreshold {
		select {
		case db.compactNotify <- struct{}{}:
		default:
		}
	}
	return nil
}

// Compact triggers manual compaction from Level 0 to Level 1.
func (db *DB) Compact() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.runCompactionLocked()
}

func (db *DB) runCompactionLocked() error {
	v := db.manifest.Current()
	if len(v.Levels[0]) == 0 {
		return nil
	}

	// Close readers of tables that will be compacted
	for _, t := range v.Levels[0] {
		if r, ok := db.tableReaders[t.FileNum]; ok {
			_ = r.Close()
			delete(db.tableReaders, t.FileNum)
		}
	}
	for _, t := range v.Levels[1] {
		if r, ok := db.tableReaders[t.FileNum]; ok {
			_ = r.Close()
			delete(db.tableReaders, t.FileNum)
		}
	}

	if err := compactor.CompactLevel(db.dir, db.manifest, 0, 1, 2*1024*1024); err != nil {
		return err
	}

	// Reopen readers for new tables
	newV := db.manifest.Current()
	for _, t := range newV.Levels[1] {
		if _, ok := db.tableReaders[t.FileNum]; !ok {
			p := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", t.FileNum))
			r, err := sstable.OpenReader(p)
			if err == nil {
				db.tableReaders[t.FileNum] = r
			}
		}
	}
	return nil
}

func (db *DB) compactionWorker() {
	defer db.bgWg.Done()
	for {
		select {
		case <-db.closeNotify:
			return
		case <-db.compactNotify:
			db.mu.Lock()
			_ = db.runCompactionLocked()
			db.mu.Unlock()
		}
	}
}

// Close gracefully flushes memory to disk, stops workers, and releases file descriptors.
func (db *DB) Close() error {
	db.mu.Lock()
	if db.isClosed {
		db.mu.Unlock()
		return nil
	}
	db.isClosed = true
	close(db.closeNotify)
	db.mu.Unlock()

	db.bgWg.Wait()

	db.mu.Lock()
	defer db.mu.Unlock()

	_ = db.flushLocked()
	_ = db.wal.Close()

	for _, r := range db.tableReaders {
		_ = r.Close()
	}
	db.tableReaders = nil
	return nil
}
