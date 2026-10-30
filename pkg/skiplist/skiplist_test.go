package skiplist

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

func TestSkipListCRUD(t *testing.T) {
	sl := New()

	sl.Put([]byte("key-b"), []byte("val-b"), false)
	sl.Put([]byte("key-a"), []byte("val-a"), false)
	sl.Put([]byte("key-c"), []byte("val-c"), false)

	val, tombstone, found := sl.Get([]byte("key-b"))
	if !found || tombstone || !bytes.Equal(val, []byte("val-b")) {
		t.Fatalf("expected val-b, got %s, found=%v, tombstone=%v", val, found, tombstone)
	}

	// Overwrite with tombstone
	sl.Put([]byte("key-b"), nil, true)
	_, tombstone, found = sl.Get([]byte("key-b"))
	if !found || !tombstone {
		t.Fatalf("expected key-b to be marked tombstone, got found=%v, tombstone=%v", found, tombstone)
	}

	// Iterate in sorted order
	it := sl.NewIterator()
	var keys []string
	for it.Next() {
		keys = append(keys, string(it.Key()))
	}

	expected := []string{"key-a", "key-b", "key-c"}
	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}
	for i, k := range keys {
		if k != expected[i] {
			t.Fatalf("expected key %s at %d, got %s", expected[i], i, k)
		}
	}
}

func TestSkipListConcurrent(t *testing.T) {
	sl := New()
	var wg sync.WaitGroup
	workers := 10
	itemsPerWorker := 500

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerWorker; i++ {
				k := []byte(fmt.Sprintf("w%02d-k%04d", workerID, i))
				v := []byte(fmt.Sprintf("val-%d", i))
				sl.Put(k, v, false)
			}
		}(w)
	}
	wg.Wait()

	if sl.Len() != workers*itemsPerWorker {
		t.Fatalf("expected len %d, got %d", workers*itemsPerWorker, sl.Len())
	}
}
