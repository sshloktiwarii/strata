package wal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWALReplayClean(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wal_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	walPath := filepath.Join(tmpDir, "test.wal")
	w, err := Open(walPath)
	if err != nil {
		t.Fatal(err)
	}

	count := 100
	for i := 0; i < count; i++ {
		k := []byte(fmt.Sprintf("key-%04d", i))
		v := []byte(fmt.Sprintf("val-%04d", i))
		if err := w.Write(TypePut, k, v); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()

	records, err := Replay(walPath)
	if err != nil {
		t.Fatalf("failed to replay: %v", err)
	}
	if len(records) != count {
		t.Fatalf("expected %d records, got %d", count, len(records))
	}
	for i, r := range records {
		expectedK := []byte(fmt.Sprintf("key-%04d", i))
		if !bytes.Equal(r.Key, expectedK) {
			t.Fatalf("record %d key mismatch: expected %s, got %s", i, expectedK, r.Key)
		}
	}
}

func TestWALCrashTruncation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wal_crash_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	walPath := filepath.Join(tmpDir, "crash.wal")
	w, err := Open(walPath)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 20; i++ {
		_ = w.Write(TypePut, []byte(fmt.Sprintf("k%d", i)), []byte(fmt.Sprintf("v%d", i)))
	}
	w.Close()

	// Append corrupt garbage bytes simulating partial write during power loss
	f, err := os.OpenFile(walPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x00, 0x00}) // incomplete frame
	f.Close()

	records, err := Replay(walPath)
	if err != nil {
		t.Fatalf("replay failed on corrupted WAL: %v", err)
	}
	if len(records) != 20 {
		t.Fatalf("expected 20 valid records before corruption point, got %d", len(records))
	}
}
