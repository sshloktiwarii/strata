# LSM-Tree Database Engine

A high-performance, crash-resilient Log-Structured Merge-tree (LSM) key-value storage engine engineered in Go with immutable on-disk SSTables, bit-vector Bloom filters, leveled background compaction, CRC32 write-ahead logging (WAL), and an embedded real-time inspection workbench.

---

## Architecture Overview

```
                      ┌──────────────────────┐
                      │    Write Request     │
                      └──────────┬───────────┘
                                 │
                 ┌───────────────┴───────────────┐
                 ▼                               ▼
       ┌───────────────────┐           ┌───────────────────┐
       │  Write-Ahead Log  │           │ In-Memory MemTable│
       │  (CRC32 Checksum) │           │    (SkipList)     │
       └───────────────────┘           └─────────┬─────────┘
                                                 │ Flush at threshold (2MB)
                                                 ▼
                                       ┌───────────────────┐
                                       │  Level 0 SSTables │
                                       │ (Overlapping keys)│
                                       └─────────┬─────────┘
                                                 │ Leveled Compactor
                                                 ▼
                                       ┌───────────────────┐
                                       │  Level 1 SSTables │
                                       │ (Partitioned / NF)│
                                       └─────────┬─────────┘
                                                 │ Multi-way merge
                                                 ▼
                                       ┌───────────────────┐
                                       │  Level 2+ Storage │
                                       │ (Cold / Archived) │
                                       └───────────────────┘
```

### Core Subsystems

1. **In-Memory MemTable (`pkg/skiplist`)**:
   - Probabilistic SkipList with $O(\log n)$ search, insert, and sequential range iteration.
   - Concurrency safety backed by `sync.RWMutex`.
   - Dynamic memory accounting with automatic flush triggering when byte size exceeds threshold (default 2MB).

2. **Append-Only Write-Ahead Log (`pkg/wal`)**:
   - CRC32-Castagnoli checksum frame framing: `[CRC32: 4B][Type: 1B][KeyLen: 4B][ValLen: 4B][Key][Value]`.
   - Zero-data-loss replay on startup.
   - Resilient crash recovery: handles mid-write power failure by detecting corrupt/partial frames and truncating at the last valid byte offset.

3. **Immutable On-Disk SSTables (`pkg/sstable`)**:
   - 4KB data blocks ending with CRC32 integrity checks.
   - Double-hashed bit-vector Bloom filter embedded in the file footer for $O(1)$ negative rejection, avoiding unnecessary disk reads.
   - Two-level sparse block index mapping block boundaries for binary search lookups.
   - 48-byte fixed metadata footer with magic identifier (`0x4C534D5452454531`).

4. **Leveled Compactor (`pkg/compactor`)**:
   - Min-heap multi-way merge sort across overlapping SSTables.
   - Promotes Level 0 tables into partitioned, non-overlapping Level 1 SSTables.
   - Purges duplicate versions and obsolete deletion tombstones at higher levels.
   - Background worker notified asynchronously upon memtable flushes.

5. **VersionSet & Manifest (`pkg/manifest`)**:
   - Transactional manifest recording active SSTables, generation IDs, and file allocations.
   - Atomic commits via temp file rename pattern ensuring corruption-free metadata states.

---

## Benchmark & Performance Telemetry

Benchmarked on Apple Silicon (M-series / ARM64):

| Workload | Throughput | Latency | Read Amplification |
| :--- | :--- | :--- | :--- |
| **Sequential Writes** | **403,773 ops/sec** (48.1 MB/s) | ~2.4 µs/op | — |
| **Random Writes (Cached)** | **499,057 ops/sec** | ~2.0 µs/op | — |
| **Random Reads (Hit Path)** | **624,792 ops/sec** | ~1.6 µs/op | **1.000** |
| **Random Reads (Bloom Negative)** | **4,863,091 ops/sec** | ~0.2 µs/op | **0.000** (Zero Disk I/O) |

---

## Crash-Recovery Fuzz Testing

The automated pressure suite includes a **50,000 keys crash-recovery fuzz test** (`pkg/lsm/crash_fuzz_test.go`):
1. Streams continuous writes to the engine.
2. Simulates an abrupt crash mid-write (power cut / SIGKILL) by abruptly terminating the process without a graceful memory flush.
3. Injects corrupted trailing bytes simulating incomplete kernel buffer writes.
4. Boots a fresh engine instance, executes WAL recovery, and asserts **100% data integrity** across all pre-crash keys.

---

## Embedded Telemetry Workbench

The engine embeds a single-page inspection workbench running on `http://localhost:5001`:

```bash
# Start server with live workbench
go run cmd/lsm-server/main.go --port 5001 --dir ./data
```

Navigate to [http://localhost:5001](http://localhost:5001) in your browser:
- **Real-Time Telemetry**: Active MemTable memory gauge, ops/sec throughput, read amplification, and Bloom filter efficiency.
- **Visual Storage Hierarchy**: Live card layout of MemTable, Level 0, Level 1, and Level 2 SSTables with byte sizes, entry counts, and key ranges.
- **Interactive KV Console**: Execute point `Put`, `Get`, and `Delete` requests with instant feedback.
- **Live Pressure Runner**: Launch 5,000 to 50,000 operation benchmarks directly from the UI with real-time diagnostic logs.

---

## Command Line Usage

### Run Automated Tests
```bash
go test -v ./...
```

### Run Crash-Recovery Fuzz Test
```bash
go test -v -run TestCrashRecoveryFuzz ./pkg/lsm
```

### Run Benchmarks
```bash
# Go benchmark harness
go test -bench=. -benchmem ./pkg/lsm

# Standalone CLI stress tester
go run cmd/lsm-bench/main.go -n 50000 -dir ./bench_data
```
