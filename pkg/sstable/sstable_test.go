package sstable

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSSTableBuildAndRead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sstable_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sstPath := filepath.Join(tmpDir, "000001.sst")
	b, err := NewBuilder(sstPath, 1000, 1024)
	if err != nil {
		t.Fatal(err)
	}

	total := 500
	for i := 0; i < total; i++ {
		k := []byte(fmt.Sprintf("user-%05d", i))
		v := []byte(fmt.Sprintf("profile-data-%05d", i))
		isTomb := (i%10 == 0) // some tombstones
		if err := b.Add(k, v, isTomb); err != nil {
			t.Fatal(err)
		}
	}

	if err := b.Finish(); err != nil {
		t.Fatal(err)
	}

	r, err := OpenReader(sstPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if r.EntryCount() != uint64(total) {
		t.Fatalf("expected %d entries, got %d", total, r.EntryCount())
	}

	// Test point lookup
	for i := 0; i < total; i++ {
		k := []byte(fmt.Sprintf("user-%05d", i))
		v, isTomb, found, err := r.Get(k)
		if err != nil {
			t.Fatalf("error getting %s: %v", k, err)
		}
		if !found {
			t.Fatalf("expected key %s to be found", k)
		}
		expectedTomb := (i%10 == 0)
		if isTomb != expectedTomb {
			t.Fatalf("key %s tombstone mismatch: expected %v, got %v", k, expectedTomb, isTomb)
		}
		expectedVal := []byte(fmt.Sprintf("profile-data-%05d", i))
		if !bytes.Equal(v, expectedVal) {
			t.Fatalf("key %s value mismatch: expected %s, got %s", k, expectedVal, v)
		}
	}

	// Test missing key (Bloom fast rejection)
	missingKey := []byte("user-99999")
	_, _, found, err := r.Get(missingKey)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatalf("expected missing key not to be found")
	}

	// Test sequential iterator
	it, err := r.NewIterator()
	if err != nil {
		t.Fatal(err)
	}

	iterCount := 0
	for it.Valid() {
		expectedK := []byte(fmt.Sprintf("user-%05d", iterCount))
		if !bytes.Equal(it.Key(), expectedK) {
			t.Fatalf("iter %d key mismatch: expected %s, got %s", iterCount, expectedK, it.Key())
		}
		iterCount++
		it.Next()
	}

	if iterCount != total {
		t.Fatalf("expected iterator count %d, got %d", total, iterCount)
	}
}
