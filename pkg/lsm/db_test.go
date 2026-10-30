package lsm

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestDBBasics(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lsm_db_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	opts := DefaultOptions(tmpDir)
	opts.MemTableThreshold = 4096 // 4KB flush threshold for fast test triggers

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	// Write 100 keys
	for i := 0; i < 100; i++ {
		k := []byte(fmt.Sprintf("user-%04d", i))
		v := []byte(fmt.Sprintf("val-%04d", i))
		if err := db.Put(k, v); err != nil {
			t.Fatalf("failed to put: %v", err)
		}
	}

	// Read before flush
	val, err := db.Get([]byte("user-0042"))
	if err != nil || !bytes.Equal(val, []byte("val-0042")) {
		t.Fatalf("expected val-0042, got %s, err=%v", val, err)
	}

	// Force Flush
	if err := db.Flush(); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}

	// Read after flush from SSTable
	val, err = db.Get([]byte("user-0042"))
	if err != nil || !bytes.Equal(val, []byte("val-0042")) {
		t.Fatalf("expected val-0042 from sstable, got %s, err=%v", val, err)
	}

	// Delete key
	if err := db.Delete([]byte("user-0042")); err != nil {
		t.Fatal(err)
	}

	// Should return ErrNotFound
	_, err = db.Get([]byte("user-0042"))
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen DB and verify persistence
	db2, err := Open(opts)
	if err != nil {
		t.Fatalf("failed to reopen db: %v", err)
	}
	defer db2.Close()

	val, err = db2.Get([]byte("user-0050"))
	if err != nil || !bytes.Equal(val, []byte("val-0050")) {
		t.Fatalf("reopened db failed to find user-0050: got %s, err=%v", val, err)
	}

	_, err = db2.Get([]byte("user-0042"))
	if err != ErrNotFound {
		t.Fatalf("expected user-0042 to remain deleted after reopen, got %v", err)
	}
}

func TestDBCompaction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lsm_compact_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	opts := DefaultOptions(tmpDir)
	opts.MemTableThreshold = 2048
	opts.L0CompactionThreshold = 3

	db, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Write enough to trigger multiple flushes
	for i := 0; i < 300; i++ {
		k := []byte(fmt.Sprintf("item-%04d", i))
		v := []byte(fmt.Sprintf("content-%04d", i))
		_ = db.Put(k, v)
	}
	_ = db.Flush()

	if err := db.Compact(); err != nil {
		t.Fatalf("manual compaction failed: %v", err)
	}

	stats := db.Stats()
	if stats.Level1Tables == 0 {
		t.Fatalf("expected level 1 tables after compaction")
	}

	// Verify all keys readable post-compaction
	for i := 0; i < 300; i++ {
		k := []byte(fmt.Sprintf("item-%04d", i))
		v, err := db.Get(k)
		if err != nil {
			t.Fatalf("missing key %s after compaction: %v", k, err)
		}
		expected := []byte(fmt.Sprintf("content-%04d", i))
		if !bytes.Equal(v, expected) {
			t.Fatalf("value mismatch for %s: expected %s, got %s", k, expected, v)
		}
	}
}
