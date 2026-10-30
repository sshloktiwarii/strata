package manifest

import (
	"os"
	"testing"
)

func TestManifestPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manifest_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m, err := Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	fn1 := m.NextFileNumber()
	fn2 := m.NextFileNumber()

	edit := &VersionEdit{
		Added: []TableMetadata{
			{FileNum: fn1, Level: 0, Size: 1024, MinKey: []byte("a"), MaxKey: []byte("m")},
			{FileNum: fn2, Level: 0, Size: 2048, MinKey: []byte("n"), MaxKey: []byte("z")},
		},
	}
	if err := m.Apply(edit); err != nil {
		t.Fatal(err)
	}

	// Re-open from disk and verify
	m2, err := Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	v := m2.Current()
	if len(v.Levels[0]) != 2 {
		t.Fatalf("expected 2 tables in level 0, got %d", len(v.Levels[0]))
	}

	// Apply deletion edit
	delEdit := &VersionEdit{
		Deleted: []TableMetadata{{FileNum: fn1, Level: 0}},
	}
	if err := m2.Apply(delEdit); err != nil {
		t.Fatal(err)
	}

	m3, _ := Open(tmpDir)
	if len(m3.Current().Levels[0]) != 1 {
		t.Fatalf("expected 1 table remaining, got %d", len(m3.Current().Levels[0]))
	}
}
