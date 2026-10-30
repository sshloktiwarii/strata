package lsm

import "sync/atomic"

// EngineStats captures runtime telemetry and amplification metrics.
type EngineStats struct {
	TotalPuts           uint64  `json:"total_puts"`
	TotalDeletes        uint64  `json:"total_deletes"`
	TotalGets           uint64  `json:"total_gets"`
	MemTableHits        uint64  `json:"memtable_hits"`
	SSTableHits         uint64  `json:"sstable_hits"`
	KeyNotFoundCount    uint64  `json:"key_not_found_count"`
	BloomFilterSaves    uint64  `json:"bloom_filter_saves"`
	SSTablesExamined    uint64  `json:"sstables_examined"`
	ReadAmplification   float64 `json:"read_amplification"`
	ActiveMemTableBytes int64   `json:"active_memtable_bytes"`
	ActiveMemTableKeys  int     `json:"active_memtable_keys"`
	Level0Tables        int     `json:"level0_tables"`
	Level1Tables        int     `json:"level1_tables"`
	Level2Tables        int     `json:"level2_tables"`
	TotalDiskBytes      int64   `json:"total_disk_bytes"`
}

// TableView represents a human-readable summary of an SSTable for inspection.
type TableView struct {
	FileNum    uint64 `json:"file_num"`
	Level      int    `json:"level"`
	Size       int64  `json:"size"`
	EntryCount uint64 `json:"entry_count"`
	MinKey     string `json:"min_key"`
	MaxKey     string `json:"max_key"`
}

type internalStats struct {
	totalPuts        uint64
	totalDeletes     uint64
	totalGets        uint64
	memtableHits     uint64
	sstableHits      uint64
	keyNotFound      uint64
	bloomSaves       uint64
	sstablesExamined uint64
}

func (s *internalStats) snapshot() EngineStats {
	gets := atomic.LoadUint64(&s.totalGets)
	examined := atomic.LoadUint64(&s.sstablesExamined)
	var readAmp float64 = 0.0
	if gets > 0 {
		readAmp = float64(examined) / float64(gets)
	}

	return EngineStats{
		TotalPuts:         atomic.LoadUint64(&s.totalPuts),
		TotalDeletes:      atomic.LoadUint64(&s.totalDeletes),
		TotalGets:         gets,
		MemTableHits:      atomic.LoadUint64(&s.memtableHits),
		SSTableHits:       atomic.LoadUint64(&s.sstableHits),
		KeyNotFoundCount:  atomic.LoadUint64(&s.keyNotFound),
		BloomFilterSaves:  atomic.LoadUint64(&s.bloomSaves),
		SSTablesExamined:  examined,
		ReadAmplification: readAmp,
	}
}
