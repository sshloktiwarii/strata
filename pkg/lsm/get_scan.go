package lsm

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sync/atomic"

	"github.com/sshloktiwarii/lsm-tree-db/pkg/sstable"
)

// Get retrieves the value associated with key.
func (db *DB) Get(key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.isClosed {
		return nil, ErrClosed
	}

	atomic.AddUint64(&db.stats.totalGets, 1)

	// 1. Search Active MemTable
	val, isTomb, found := db.memTable.Get(key)
	if found {
		if isTomb {
			atomic.AddUint64(&db.stats.keyNotFound, 1)
			return nil, ErrNotFound
		}
		atomic.AddUint64(&db.stats.memtableHits, 1)
		return val, nil
	}

	v := db.manifest.Current()

	// 2. Search Level 0 (Newest to Oldest)
	l0 := v.Levels[0]
	for i := len(l0) - 1; i >= 0; i-- {
		t := l0[i]
		r, ok := db.tableReaders[t.FileNum]
		if !ok {
			p := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", t.FileNum))
			var err error
			r, err = sstable.OpenReader(p)
			if err != nil {
				continue
			}
		}

		atomic.AddUint64(&db.stats.sstablesExamined, 1)
		if !r.MayContain(key) {
			atomic.AddUint64(&db.stats.bloomSaves, 1)
			continue
		}

		v, tomb, found, err := r.Get(key)
		if err == nil && found {
			if tomb {
				atomic.AddUint64(&db.stats.keyNotFound, 1)
				return nil, ErrNotFound
			}
			atomic.AddUint64(&db.stats.sstableHits, 1)
			return v, nil
		}
	}

	// 3. Search Level 1+ (Non-overlapping partitions)
	for lvl := 1; lvl <= 5; lvl++ {
		tables := v.Levels[lvl]
		for _, t := range tables {
			if len(t.MinKey) > 0 && bytes.Compare(key, t.MinKey) < 0 {
				continue
			}
			if len(t.MaxKey) > 0 && bytes.Compare(key, t.MaxKey) > 0 {
				continue
			}

			r, ok := db.tableReaders[t.FileNum]
			if !ok {
				p := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", t.FileNum))
				var err error
				r, err = sstable.OpenReader(p)
				if err != nil {
					continue
				}
			}

			atomic.AddUint64(&db.stats.sstablesExamined, 1)
			if !r.MayContain(key) {
				atomic.AddUint64(&db.stats.bloomSaves, 1)
				continue
			}

			v, tomb, found, err := r.Get(key)
			if err == nil && found {
				if tomb {
					atomic.AddUint64(&db.stats.keyNotFound, 1)
					return nil, ErrNotFound
				}
				atomic.AddUint64(&db.stats.sstableHits, 1)
				return v, nil
			}
		}
	}

	atomic.AddUint64(&db.stats.keyNotFound, 1)
	return nil, ErrNotFound
}

// Stats returns a snapshot of database runtime statistics and read amplification.
func (db *DB) Stats() EngineStats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	s := db.stats.snapshot()
	s.ActiveMemTableBytes = db.memTable.ByteSize()
	s.ActiveMemTableKeys = db.memTable.Len()

	v := db.manifest.Current()
	s.Level0Tables = len(v.Levels[0])
	s.Level1Tables = len(v.Levels[1])
	s.Level2Tables = len(v.Levels[2])

	var totalDisk int64 = 0
	for _, tables := range v.Levels {
		for _, t := range tables {
			totalDisk += t.Size
		}
	}
	s.TotalDiskBytes = totalDisk
	return s
}

// InspectLevels returns a table-by-table summary across all levels.
func (db *DB) InspectLevels() []TableView {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []TableView
	v := db.manifest.Current()
	for lvl, tables := range v.Levels {
		for _, t := range tables {
			result = append(result, TableView{
				FileNum:    t.FileNum,
				Level:      lvl,
				Size:       t.Size,
				EntryCount: t.EntryCount,
				MinKey:     string(t.MinKey),
				MaxKey:     string(t.MaxKey),
			})
		}
	}
	return result
}
