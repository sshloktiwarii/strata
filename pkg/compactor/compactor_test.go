package compactor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sshloktiwarii/lsm-tree-db/pkg/manifest"
	"github.com/sshloktiwarii/lsm-tree-db/pkg/sstable"
)

func TestCompactL0ToL1(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compactor_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m, err := manifest.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create 3 L0 tables with overlapping keys
	var l0Tables []manifest.TableMetadata
	for batch := 0; batch < 3; batch++ {
		fn := m.NextFileNumber()
		p := filepath.Join(tmpDir, fmt.Sprintf("%06d.sst", fn))
		b, err := sstable.NewBuilder(p, 50, 512)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 20; i++ {
			k := []byte(fmt.Sprintf("k%03d", i))
			v := []byte(fmt.Sprintf("v-batch%d-%03d", batch, i))
			_ = b.Add(k, v, false)
		}
		_ = b.Finish()
		stat, _ := os.Stat(p)
		l0Tables = append(l0Tables, manifest.TableMetadata{
			FileNum:    fn,
			Level:      0,
			Size:       stat.Size(),
			EntryCount: 20,
			MinKey:     []byte("k000"),
			MaxKey:     []byte("k019"),
		})
	}

	_ = m.Apply(&manifest.VersionEdit{Added: l0Tables})

	// Run compaction
	if err := CompactLevel(tmpDir, m, 0, 1, 2*1024*1024); err != nil {
		t.Fatalf("compaction failed: %v", err)
	}

	v := m.Current()
	if len(v.Levels[0]) != 0 {
		t.Fatalf("expected 0 tables in level 0, got %d", len(v.Levels[0]))
	}
	if len(v.Levels[1]) == 0 {
		t.Fatalf("expected at least 1 table in level 1")
	}

	// Read merged table and verify newest batch version was preserved
	l1Path := filepath.Join(tmpDir, fmt.Sprintf("%06d.sst", v.Levels[1][0].FileNum))
	r, err := sstable.OpenReader(l1Path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	val, _, found, _ := r.Get([]byte("k005"))
	if !found {
		t.Fatalf("expected k005 to be found")
	}
	expectedVal := []byte("v-batch2-005") // latest batch wins
	if !bytes.Equal(val, expectedVal) {
		t.Fatalf("expected %s, got %s", expectedVal, val)
	}
}
