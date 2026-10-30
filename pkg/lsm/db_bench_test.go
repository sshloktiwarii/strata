package lsm

import (
	"fmt"
	"os"
	"testing"
)

func BenchmarkLSMPut(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "lsm_bench_put_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	opts := DefaultOptions(tmpDir)
	opts.MemTableThreshold = 4 * 1024 * 1024 // 4MB
	db, err := Open(opts)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	val := []byte("standard-benchmark-payload-value-128b-data-pad-00000000000000000000000000000000000000000000000000000000000000000000000000000000")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		k := []byte(fmt.Sprintf("bench-key-%09d", i))
		if err := db.Put(k, val); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLSMGet(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "lsm_bench_get_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	opts := DefaultOptions(tmpDir)
	opts.MemTableThreshold = 2 * 1024 * 1024 // 2MB
	db, err := Open(opts)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	totalPreload := 20000
	val := []byte("bench-value-content-for-reading")
	for i := 0; i < totalPreload; i++ {
		k := []byte(fmt.Sprintf("user-id-%06d", i))
		_ = db.Put(k, val)
	}
	_ = db.Flush()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		k := []byte(fmt.Sprintf("user-id-%06d", i%totalPreload))
		_, err := db.Get(k)
		if err != nil {
			b.Fatalf("failed to get: %v", err)
		}
	}
	b.StopTimer()

	stats := db.Stats()
	b.Logf("Read Amplification: %.3f (SSTables Examined: %d, Gets: %d, Bloom Saves: %d)",
		stats.ReadAmplification, stats.SSTablesExamined, stats.TotalGets, stats.BloomFilterSaves)
}
