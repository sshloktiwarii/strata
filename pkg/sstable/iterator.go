package sstable

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
)

// Iterator provides sequential iteration through all entries in an SSTable.
type Iterator struct {
	reader   *Reader
	blockIdx int
	blockBuf *bytes.Reader
	curr     Entry
	valid    bool
	err      error
}

// NewIterator creates a sequential iterator over the SSTable.
func (r *Reader) NewIterator() (*Iterator, error) {
	it := &Iterator{reader: r, blockIdx: -1}
	it.Next()
	return it, it.err
}

// Next advances to the next entry.
func (it *Iterator) Next() bool {
	if it.err != nil {
		it.valid = false
		return false
	}

	for {
		if it.blockBuf != nil && it.blockBuf.Len() > 0 {
			hdr := make([]byte, 7)
			if _, err := io.ReadFull(it.blockBuf, hdr); err != nil {
				it.err = err
				it.valid = false
				return false
			}
			kLen := binary.BigEndian.Uint16(hdr[0:2])
			vLen := binary.BigEndian.Uint32(hdr[2:6])
			isTomb := hdr[6] == 1

			k := make([]byte, kLen)
			if _, err := io.ReadFull(it.blockBuf, k); err != nil {
				it.err = err
				it.valid = false
				return false
			}
			v := make([]byte, vLen)
			if vLen > 0 {
				if _, err := io.ReadFull(it.blockBuf, v); err != nil {
					it.err = err
					it.valid = false
					return false
				}
			}

			it.curr = Entry{Key: k, Value: v, IsTombstone: isTomb}
			it.valid = true
			return true
		}

		// Advance to next block
		it.blockIdx++
		if it.blockIdx >= len(it.reader.index) {
			it.valid = false
			return false
		}

		handle := it.reader.index[it.blockIdx].Handle
		rawBlock := make([]byte, handle.Length)
		if _, err := it.reader.file.ReadAt(rawBlock, int64(handle.Offset)); err != nil {
			it.err = err
			it.valid = false
			return false
		}
		data := rawBlock[:len(rawBlock)-4]
		expectedCRC := binary.BigEndian.Uint32(rawBlock[len(rawBlock)-4:])
		if crc32.Checksum(data, crcTable) != expectedCRC {
			it.err = ErrInvalidSSTable
			it.valid = false
			return false
		}
		it.blockBuf = bytes.NewReader(data)
	}
}

// Valid returns whether the current position is valid.
func (it *Iterator) Valid() bool {
	return it.valid && it.err == nil
}

// Key returns the current entry key.
func (it *Iterator) Key() []byte {
	return it.curr.Key
}

// Value returns the current entry value.
func (it *Iterator) Value() []byte {
	return it.curr.Value
}

// IsTombstone returns whether current entry is deleted.
func (it *Iterator) IsTombstone() bool {
	return it.curr.IsTombstone
}

// Error returns any iterator error.
func (it *Iterator) Error() error {
	return it.err
}
