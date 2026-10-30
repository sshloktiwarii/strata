package wal

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

const (
	TypePut    byte = 0
	TypeDelete byte = 1
	headerSize      = 4 + 1 + 4 + 4 // CRC32 (4) + Type (1) + KeyLen (4) + ValLen (4) = 13 bytes
)

var (
	crcTable          = crc32.MakeTable(crc32.Castagnoli)
	ErrCorruptRecord  = errors.New("wal record checksum mismatch or truncated")
)

// Record represents a single write-ahead log entry.
type Record struct {
	Type  byte
	Key   []byte
	Value []byte
}

// WAL provides an append-only write-ahead log with CRC32 integrity verification.
type WAL struct {
	mu   sync.Mutex
	file *os.File
	path string
	size int64
}

// Open opens or creates a WAL file at path.
func Open(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	return &WAL{file: f, path: path, size: stat.Size()}, nil
}

// Write appends a record to the WAL.
func (w *WAL) Write(recType byte, key, value []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	kLen := uint32(len(key))
	vLen := uint32(len(value))
	payloadSize := 1 + 4 + 4 + int(kLen) + int(vLen)
	payload := make([]byte, payloadSize)

	payload[0] = recType
	binary.BigEndian.PutUint32(payload[1:5], kLen)
	binary.BigEndian.PutUint32(payload[5:9], vLen)
	copy(payload[9:9+kLen], key)
	copy(payload[9+kLen:], value)

	checksum := crc32.Checksum(payload, crcTable)
	frame := make([]byte, 4+payloadSize)
	binary.BigEndian.PutUint32(frame[0:4], checksum)
	copy(frame[4:], payload)

	n, err := w.file.Write(frame)
	if err != nil {
		return err
	}
	w.size += int64(n)
	return nil
}

// Sync forces written buffers to disk.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Sync()
}

// Close closes the WAL file descriptor.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// Replay reads all valid records from the WAL. If a partial or corrupted trailing frame is found
// (e.g. from an uncompleted crash mid-write), it truncates the WAL to the last valid byte offset.
func Replay(path string) ([]*Record, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var records []*Record
	var validOffset int64 = 0
	hdr := make([]byte, headerSize)

	for {
		_, err := io.ReadFull(f, hdr)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			break
		}

		checksum := binary.BigEndian.Uint32(hdr[0:4])
		recType := hdr[4]
		kLen := binary.BigEndian.Uint32(hdr[5:9])
		vLen := binary.BigEndian.Uint32(hdr[9:13])

		body := make([]byte, kLen+vLen)
		if _, err := io.ReadFull(f, body); err != nil {
			// Incomplete trailing body: crash recovery truncation
			break
		}

		payload := make([]byte, 1+4+4+len(body))
		payload[0] = recType
		binary.BigEndian.PutUint32(payload[1:5], kLen)
		binary.BigEndian.PutUint32(payload[5:9], vLen)
		copy(payload[9:], body)

		if crc32.Checksum(payload, crcTable) != checksum {
			// Corrupted data: stop at last known valid offset
			break
		}

		key := body[:kLen]
		val := body[kLen:]
		records = append(records, &Record{Type: recType, Key: key, Value: val})
		validOffset += int64(headerSize + len(body))
	}

	// Truncate any incomplete or corrupt trailing writes
	if err := f.Truncate(validOffset); err != nil {
		return nil, err
	}
	return records, nil
}
