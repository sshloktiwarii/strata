package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// TableMetadata holds on-disk info for a single SSTable.
type TableMetadata struct {
	FileNum    uint64 `json:"file_num"`
	Level      int    `json:"level"`
	Size       int64  `json:"size"`
	EntryCount uint64 `json:"entry_count"`
	MinKey     []byte `json:"min_key"`
	MaxKey     []byte `json:"max_key"`
}

// Version represents the active state of all levels.
type Version struct {
	Levels map[int][]TableMetadata `json:"levels"`
}

// VersionEdit specifies delta changes applied to a Version.
type VersionEdit struct {
	Added   []TableMetadata `json:"added"`
	Deleted []TableMetadata `json:"deleted"`
}

// Manifest tracks the LSM tree metadata, active tables, and version history.
type Manifest struct {
	mu           sync.RWMutex
	dir          string
	current      *Version
	nextFileNum  uint64
	manifestFile string
}

// State represents the serialized manifest structure.
type State struct {
	NextFileNum uint64                  `json:"next_file_num"`
	Levels      map[int][]TableMetadata `json:"levels"`
}

// Open loads or initializes the Manifest in dir.
func Open(dir string) (*Manifest, error) {
	manifestPath := filepath.Join(dir, "MANIFEST.json")
	m := &Manifest{
		dir:          dir,
		current:      &Version{Levels: make(map[int][]TableMetadata)},
		nextFileNum:  1,
		manifestFile: manifestPath,
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return m, m.save()
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	m.nextFileNum = state.NextFileNum
	if m.nextFileNum == 0 {
		m.nextFileNum = 1
	}
	m.current.Levels = state.Levels
	if m.current.Levels == nil {
		m.current.Levels = make(map[int][]TableMetadata)
	}
	return m, nil
}

// NextFileNumber increments and returns a unique file number.
func (m *Manifest) NextFileNumber() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	num := m.nextFileNum
	m.nextFileNum++
	_ = m.save()
	return num
}

// Apply atomically applies a VersionEdit and persists the new state.
func (m *Manifest) Apply(edit *VersionEdit) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Remove deleted tables
	for _, del := range edit.Deleted {
		tables := m.current.Levels[del.Level]
		var updated []TableMetadata
		for _, t := range tables {
			if t.FileNum != del.FileNum {
				updated = append(updated, t)
			}
		}
		m.current.Levels[del.Level] = updated
	}

	// 2. Add new tables
	for _, add := range edit.Added {
		m.current.Levels[add.Level] = append(m.current.Levels[add.Level], add)
	}

	return m.save()
}

// Current returns a snapshot of active levels.
func (m *Manifest) Current() *Version {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cp := &Version{Levels: make(map[int][]TableMetadata)}
	for lvl, tbls := range m.current.Levels {
		cp.Levels[lvl] = append([]TableMetadata(nil), tbls...)
	}
	return cp
}

func (m *Manifest) save() error {
	state := State{
		NextFileNum: m.nextFileNum,
		Levels:      m.current.Levels,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := fmt.Sprintf("%s.tmp", m.manifestFile)
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, m.manifestFile)
}
