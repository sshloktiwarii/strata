package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/sshloktiwarii/strata/pkg/lsm"
)

func main() {
	count := flag.Int("n", 50000, "number of operations per phase")
	dir := flag.String("dir", "./bench_data", "database directory")
	flag.Parse()

	_ = os.RemoveAll(*dir)
	defer os.RemoveAll(*dir)

	opts := lsm.DefaultOptions(*dir)
	opts.MemTableThreshold = 2 * 1024 * 1024 // 2MB
	db, err := lsm.Open(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("==================================================")
	fmt.Println("       LSM-TREE-DB HIGH-PRESSURE BENCHMARK        ")
	fmt.Println("==================================================")
	fmt.Printf("Operations: %d | MemTable Limit: 2MB | Block: 4KB\n\n", *count)

	val := []byte("payload-standard-128b-benchmark-data-block-testing-value-00000000000000000000000000000000000000000000000000000000000000000000")

	// Phase 1: Sequential Writes
	fmt.Print("[1/3] Benchmarking Sequential Writes... ")
	start := time.Now()
	for i := 0; i < *count; i++ {
		k := []byte(fmt.Sprintf("key-seq-%08d", i))
		if err := db.Put(k, val); err != nil {
			fmt.Printf("Put failed: %v\n", err)
			return
		}
	}
	writeDur := time.Since(start)
	writeOps := float64(*count) / writeDur.Seconds()
	fmt.Printf("DONE\n      Time: %v | Throughput: %.0f ops/sec (%.2f MB/s)\n\n",
		writeDur.Round(time.Millisecond), writeOps, (writeOps*float64(len(val)))/(1024*1024))

	// Flush active memtable & trigger compaction
	fmt.Print("Flushing MemTable and triggering Compaction... ")
	_ = db.Flush()
	_ = db.Compact()
	fmt.Println("DONE")

	// Phase 2: Random Reads (Existing Keys)
	fmt.Print("[2/3] Benchmarking Random Reads (Hit Path)... ")
	rnd := rand.New(rand.NewSource(42))
	start = time.Now()
	for i := 0; i < *count; i++ {
		target := rnd.Intn(*count)
		k := []byte(fmt.Sprintf("key-seq-%08d", target))
		_, err := db.Get(k)
		if err != nil {
			fmt.Printf("Get failed on %s: %v\n", k, err)
			return
		}
	}
	readDur := time.Since(start)
	readOps := float64(*count) / readDur.Seconds()
	fmt.Printf("DONE\n      Time: %v | Throughput: %.0f ops/sec\n\n",
		readDur.Round(time.Millisecond), readOps)

	// Phase 3: Random Reads (Non-Existent Keys - Bloom Filter Test)
	fmt.Print("[3/3] Benchmarking Non-Existent Reads (Bloom Rejection)... ")
	start = time.Now()
	for i := 0; i < *count; i++ {
		k := []byte(fmt.Sprintf("missing-rand-%09d", rnd.Intn(10000000)))
		_, _ = db.Get(k)
	}
	missingDur := time.Since(start)
	missingOps := float64(*count) / missingDur.Seconds()
	fmt.Printf("DONE\n      Time: %v | Throughput: %.0f ops/sec\n\n",
		missingDur.Round(time.Millisecond), missingOps)

	// Summary & Amplification
	stats := db.Stats()
	fmt.Println("--------------------------------------------------")
	fmt.Println("                 FINAL METRICS                    ")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Total Gets:            %d\n", stats.TotalGets)
	fmt.Printf("SSTables Probed:       %d\n", stats.SSTablesExamined)
	fmt.Printf("Bloom Filter Rejections: %d (SSTable reads saved)\n", stats.BloomFilterSaves)
	fmt.Printf("Read Amplification:    %.3f SSTables/read\n", stats.ReadAmplification)
	fmt.Printf("Level-0 SSTables:      %d\n", stats.Level0Tables)
	fmt.Printf("Level-1 SSTables:      %d\n", stats.Level1Tables)
	fmt.Printf("Total Disk Footprint:  %.2f KB\n", float64(stats.TotalDiskBytes)/1024)
	fmt.Println("==================================================")
}
