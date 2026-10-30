```
██╗     ███████╗███╗   ███╗    ████████╗██████╗ ███████╗███████╗
██║     ██╔════╝████╗ ████║    ╚══██╔══╝██╔══██╗██╔════╝██╔════╝
██║     ███████╗██╔████╔██║       ██║   ██████╔╝█████╗  █████╗  
██║     ╚════██║██║╚██╔╝██║       ██║   ██╔══██╗██╔══╝  ██╔══╝  
███████╗███████║██║ ╚═╝ ██║       ██║   ██║  ██║███████╗███████╗
╚══════╝╚══════╝╚═╝     ╚═╝       ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝
```

# LSM-TREE-DB // LOG-STRUCTURED MERGE STORAGE ENGINE

> **High-performance Log-Structured Merge-tree (LSM) key-value storage engine in Go with immutable on-disk SSTables, bit-vector Bloom filters, leveled background compaction, CRC32 WAL, and embedded inspection workbench.**

[![Go Version](https://img.shields.io/badge/Language-Go%201.22%2B-blue.svg?style=flat-square&logo=go)](https://golang.org)
[![Storage](https://img.shields.io/badge/Architecture-Leveled%20LSM--Tree-cyan.svg?style=flat-square)](#architecture-overview)
[![Integrity](https://img.shields.io/badge/Integrity-CRC32--Castagnoli-brightgreen.svg?style=flat-square)](pkg/wal/)
[![Workbench](https://img.shields.io/badge/Workbench-Port%205001-orange.svg?style=flat-square)](http://localhost:5001)
[![License: MIT](https://img.shields.io/badge/License-MIT-gray.svg?style=flat-square)](LICENSE)

---

## 📌 Repository Metadata

* **About**: Production-grade Go LSM-Tree key-value database engine with leveled compaction and real-time visual inspection workbench.
* **Description**: Delivers sub-millisecond writes and predictable reads under heavy ingestion. Combines a probabilistic SkipList MemTable with crash-resilient CRC32 Write-Ahead Logging (WAL), immutable binary SSTables with sparse index blocks, Murmur3 Bloom filters for zero-I/O negative lookups, and an embedded browser telemetry workbench.
* **Topics & Tags**: `golang`, `lsm-tree`, `storage-engine`, `database`, `sstable`, `bloom-filter`, `wal`, `skiplist`, `compaction`, `swiss-brutalism`, `embedded-database`

---

## 🏛️ Architecture Overview

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

---

## ⚙️ Core Subsystem Invariants

| Subsystem | File & Module | Algorithm & Data Structure | Invariant Guarantee |
| :--- | :--- | :--- | :--- |
| **MemTable** | `pkg/skiplist` | Probabilistic SkipList ($O(\log n)$ concurrent reads/writes) | Thread-safe `sync.RWMutex` with deterministic byte accounting and 2MB flush threshold. |
| **WAL Engine** | `pkg/wal` | Append-only log with CRC32-Castagnoli framed envelopes | **Zero-Loss Crash Recovery**: Corrupted or torn writes detected and truncated safely on boot. |
| **SSTable Layer** | `pkg/sstable` | Immutable binary files with trailer footer and sparse index blocks | Single-seek point lookups; binary search within sparse block offsets ($4\text{KB}$ strides). |
| **Bloom Filters** | `pkg/bloom` | Bit-vector filter with 7 optimal hash functions (Murmur3) | **Zero Disk I/O on Cache Miss**: 99% negative lookup rejection rate without disk reads. |
| **Compactor** | `pkg/compactor`| Multi-way $K$-merge leveled compaction worker | Background thread garbage-collects tombstones and merges overlapping SSTables into $L1/L2$. |

---

## ⚡ Quickstart & Verification

```bash
# Navigate to project directory
cd fleet-projects/lsm-tree-db

# Run full test suite including crash-recovery and compaction benchmarks
go test -v -race ./...

# Launch LSM database server on port 5001 with embedded workbench
go run ./cmd/lsm-server -port 5001 -dir ./data
```

Open **[http://localhost:5001](http://localhost:5001)** to launch the real-time LSM-Tree visualizer and interactive key-value inspector.
