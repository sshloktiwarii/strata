package lsm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCrashRecoveryFuzz writes 50,000 keys, simulates a sudden crash mid-write
// with corrupt trailing bytes (like sudden power cut during IO), restarts the engine,
// and asserts 100% data integrity from WAL replay.
func TestCrashRecoveryFuzz(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lsm_crash_fuzz_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	opts := DefaultOptions(tmpDir)
	// High threshold to ensure WAL holds active writes before flush
	opts.MemTableThreshold = 64 * 1024 * 1024
	opts.SyncWAL = false

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("failed to open initial db: %v", err)
	}

	const totalKeys = 50000
	const crashAt = 31250 // Simulate sudden crash after writing this many keys

	t.Logf("Writing %d keys before simulating sudden crash...", crashAt)
	for i := 0; i < crashAt; i++ {
		k := []byte(fmt.Sprintf("fuzz-key-%07d", i))
		v := []byte(fmt.Sprintf("fuzz-payload-val-%07d-payload-data", i))
		if err := db.Put(k, v); err != nil {
			t.Fatalf("failed to put key %d: %v", i, err)
		}
	}

	// Force fsync of WAL to ensure disk has the bytes
	if err := db.wal.Sync(); err != nil {
		t.Fatalf("failed to sync wal: %v", err)
	}

	// Abruptly simulate hard crash:
	// 1. Close WAL file descriptor directly WITHOUT flushing MemTable to SSTables
	// 2. Append corrupted trailing bytes simulating incomplete write during power cut
	_ = db.wal.Close()

	walPath := filepath.Join(tmpDir, "active.wal")
	f, err := os.OpenFile(walPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open wal for injection: %v", err)
	}
	// Incomplete frame bytes (corrupt header and partial body)
	corruptBytes := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x00, 0x05, 0x00, 0x00}
	if _, err := f.Write(corruptBytes); err != nil {
		t.Fatal(err)
	}
	f.Close()

	t.Log("Simulated sudden power cut / crash mid-write. Restarting engine...")

	// Restart engine
	db2, err := Open(opts)
	if err != nil {
		t.Fatalf("engine failed to recover from WAL crash: %v", err)
	}
	defer db2.Close()

	t.Logf("Engine recovered cleanly. Asserting 100%% integrity across all %d pre-crash keys...", crashAt)

	// Verify all pre-crash keys exist with 100% data integrity
	for i := 0; i < crashAt; i++ {
		k := []byte(fmt.Sprintf("fuzz-key-%07d", i))
		v, err := db2.Get(k)
		if err != nil {
			t.Fatalf("missing key %s after recovery: %v", k, err)
		}
		expectedV := []byte(fmt.Sprintf("fuzz-payload-val-%07d-payload-data", i))
		if !bytes.Equal(v, expectedV) {
			t.Fatalf("corruption in key %s: expected %s, got %s", k, expectedV, v)
		}
	}

	// Verify keys after crash point do NOT exist
	for i := crashAt; i < crashAt+100; i++ {
		k := []byte(fmt.Sprintf("fuzz-key-%07d", i))
		_, err := db2.Get(k)
		if err != ErrNotFound {
			t.Fatalf("key %s after crash point should not exist, got err=%v", k, err)
		}
	}

	t.Log("100% data integrity verified. Writing remaining keys to reach 50,000 total...")
	for i := crashAt; i < totalKeys; i++ {
		k := []byte(fmt.Sprintf("fuzz-key-%07d", i))
		v := []byte(fmt.Sprintf("fuzz-payload-val-%07d-payload-data", i))
		if err := db2.Put(k, v); err != nil {
			t.Fatalf("failed to write post-crash key %d: %v", i, err)
		}
	}

	// Verify final state
	for i := 0; i < totalKeys; i++ {
		k := []byte(fmt.Sprintf("fuzz-key-%07d", i))
		v, err := db2.Get(k)
		if err != nil {
			t.Fatalf("missing key %s in final check: %v", k, err)
		}
		expectedV := []byte(fmt.Sprintf("fuzz-payload-val-%07d-payload-data", i))
		if !bytes.Equal(v, expectedV) {
			t.Fatalf("corrupted key %s in final check", k)
		}
	}

	t.Logf("Success: 50,000 keys fuzz & crash-recovery passed with 100%% data integrity!")
}
