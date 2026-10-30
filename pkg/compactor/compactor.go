package compactor

import (
	"bytes"
	"container/heap"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sshloktiwarii/lsm-tree-db/pkg/manifest"
	"github.com/sshloktiwarii/lsm-tree-db/pkg/sstable"
)

type iterCursor struct {
	iter     *sstable.Iterator
	priority int // Higher value = newer table (takes precedence on duplicate key)
	index    int
}

type cursorHeap []*iterCursor

func (h cursorHeap) Len() int           { return len(h) }
func (h cursorHeap) Less(i, j int) bool {
	cmp := bytes.Compare(h[i].iter.Key(), h[j].iter.Key())
	if cmp == 0 {
		return h[i].priority > h[j].priority // newer priority first
	}
	return cmp < 0
}
func (h cursorHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *cursorHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*iterCursor)
	item.index = n
	*h = append(*h, item)
}
func (h *cursorHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]
	return item
}

// CompactLevel merges tables from sourceLevel into destLevel.
func CompactLevel(dir string, m *manifest.Manifest, sourceLevel, destLevel int, targetFileSize int64) error {
	v := m.Current()
	sourceTables := v.Levels[sourceLevel]
	if len(sourceTables) == 0 {
		return nil
	}

	destTables := v.Levels[destLevel]

	var readers []*sstable.Reader
	defer func() {
		for _, r := range readers {
			_ = r.Close()
		}
	}()

	var h cursorHeap
	heap.Init(&h)

	var toDelete []manifest.TableMetadata
	toDelete = append(toDelete, sourceTables...)

	// Open source level readers
	for i, t := range sourceTables {
		path := filepath.Join(dir, fmt.Sprintf("%06d.sst", t.FileNum))
		r, err := sstable.OpenReader(path)
		if err != nil {
			return err
		}
		readers = append(readers, r)
		it, err := r.NewIterator()
		if err != nil {
			return err
		}
		if it.Valid() {
			heap.Push(&h, &iterCursor{iter: it, priority: 1000 + i})
		}
	}

	// Open overlapping dest level readers
	for _, t := range destTables {
		path := filepath.Join(dir, fmt.Sprintf("%06d.sst", t.FileNum))
		r, err := sstable.OpenReader(path)
		if err != nil {
			continue
		}
		readers = append(readers, r)
		it, err := r.NewIterator()
		if err != nil {
			continue
		}
		if it.Valid() {
			heap.Push(&h, &iterCursor{iter: it, priority: int(t.FileNum)})
			toDelete = append(toDelete, t)
		}
	}

	if h.Len() == 0 {
		return nil
	}

	var addedTables []manifest.TableMetadata
	var curBuilder *sstable.Builder
	var curFileNum uint64
	var curMinKey, curMaxKey []byte
	var curEntryCount uint64
	var curPath string

	finishCurrentBuilder := func() error {
		if curBuilder == nil {
			return nil
		}
		if err := curBuilder.Finish(); err != nil {
			return err
		}
		stat, _ := os.Stat(curPath)
		var size int64
		if stat != nil {
			size = stat.Size()
		}
		addedTables = append(addedTables, manifest.TableMetadata{
			FileNum:    curFileNum,
			Level:      destLevel,
			Size:       size,
			EntryCount: curEntryCount,
			MinKey:     curMinKey,
			MaxKey:     curMaxKey,
		})
		curBuilder = nil
		return nil
	}

	var lastKey []byte

	for h.Len() > 0 {
		top := heap.Pop(&h).(*iterCursor)
		key := top.iter.Key()
		val := top.iter.Value()
		isTomb := top.iter.IsTombstone()

		// Skip duplicate keys from older iterators
		isDuplicate := (len(lastKey) > 0 && bytes.Equal(lastKey, key))
		if !isDuplicate {
			lastKey = bytes.Clone(key)

			// Purge tombstones at destination level 2+
			if !(destLevel >= 2 && isTomb) {
				if curBuilder == nil {
					curFileNum = m.NextFileNumber()
					curPath = filepath.Join(dir, fmt.Sprintf("%06d.sst", curFileNum))
					var err error
					curBuilder, err = sstable.NewBuilder(curPath, 10000, 4096)
					if err != nil {
						return err
					}
					curMinKey = bytes.Clone(key)
					curEntryCount = 0
				}

				if err := curBuilder.Add(key, val, isTomb); err != nil {
					return err
				}
				curMaxKey = bytes.Clone(key)
				curEntryCount++

				// Check if reached target file size threshold
				stat, _ := os.Stat(curPath)
				if stat != nil && stat.Size() >= targetFileSize {
					if err := finishCurrentBuilder(); err != nil {
						return err
					}
				}
			}
		}

		if top.iter.Next() && top.iter.Valid() {
			heap.Push(&h, top)
		}
	}

	if err := finishCurrentBuilder(); err != nil {
		return err
	}

	// Sort dest tables by minKey for non-overlapping ordered traversal
	sort.Slice(addedTables, func(i, j int) bool {
		return bytes.Compare(addedTables[i].MinKey, addedTables[j].MinKey) < 0
	})

	// Apply manifest edit
	edit := &manifest.VersionEdit{
		Added:   addedTables,
		Deleted: toDelete,
	}
	if err := m.Apply(edit); err != nil {
		return err
	}

	// Clean up merged SSTable files
	for _, del := range toDelete {
		p := filepath.Join(dir, fmt.Sprintf("%06d.sst", del.FileNum))
		_ = os.Remove(p)
	}

	return nil
}
