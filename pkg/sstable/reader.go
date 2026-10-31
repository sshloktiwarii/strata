package sstable

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"sort"

	"github.com/sshloktiwarii/strata/pkg/bloom"
)

// Reader provides read access to an on-disk SSTable.
type Reader struct {
	file       *os.File
	path       string
	footer     Footer
	index      []IndexEntry
	bloom      *bloom.BloomFilter
	fileSize   int64
	minKey     []byte
	maxKey     []byte
}

// OpenReader opens an SSTable and reads footer, bloom filter, and sparse index into memory.
func OpenReader(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if stat.Size() < FooterSize {
		f.Close()
		return nil, ErrInvalidSSTable
	}

	// 1. Read footer
	footerBuf := make([]byte, FooterSize)
	if _, err := f.ReadAt(footerBuf, stat.Size()-FooterSize); err != nil {
		f.Close()
		return nil, err
	}

	footer := Footer{
		IndexHandle: BlockHandle{
			Offset: binary.BigEndian.Uint64(footerBuf[0:8]),
			Length: binary.BigEndian.Uint64(footerBuf[8:16]),
		},
		BloomHandle: BlockHandle{
			Offset: binary.BigEndian.Uint64(footerBuf[16:24]),
			Length: binary.BigEndian.Uint64(footerBuf[24:32]),
		},
		EntryCount: binary.BigEndian.Uint64(footerBuf[32:40]),
		Magic:      binary.BigEndian.Uint64(footerBuf[40:48]),
	}

	if footer.Magic != MagicNumber {
		f.Close()
		return nil, ErrInvalidSSTable
	}

	// 2. Read Bloom Filter
	bloomBuf := make([]byte, footer.BloomHandle.Length)
	if _, err := f.ReadAt(bloomBuf, int64(footer.BloomHandle.Offset)); err != nil {
		f.Close()
		return nil, err
	}
	bf, err := bloom.Decode(bloomBuf)
	if err != nil {
		f.Close()
		return nil, err
	}

	// 3. Read Sparse Index
	indexBuf := make([]byte, footer.IndexHandle.Length)
	if _, err := f.ReadAt(indexBuf, int64(footer.IndexHandle.Offset)); err != nil {
		f.Close()
		return nil, err
	}

	var index []IndexEntry
	var offset int = 0
	for offset < len(indexBuf) {
		if offset+2 > len(indexBuf) {
			break
		}
		kLen := int(binary.BigEndian.Uint16(indexBuf[offset : offset+2]))
		offset += 2
		if offset+kLen+16 > len(indexBuf) {
			break
		}
		key := make([]byte, kLen)
		copy(key, indexBuf[offset:offset+kLen])
		offset += kLen
		bOffset := binary.BigEndian.Uint64(indexBuf[offset : offset+8])
		bLength := binary.BigEndian.Uint64(indexBuf[offset+8 : offset+16])
		offset += 16

		index = append(index, IndexEntry{
			Key:    key,
			Handle: BlockHandle{Offset: bOffset, Length: bLength},
		})
	}

	r := &Reader{
		file:     f,
		path:     path,
		footer:   footer,
		index:    index,
		bloom:    bf,
		fileSize: stat.Size(),
	}
	if len(index) > 0 {
		r.minKey = index[0].Key
	}
	return r, nil
}

// MayContain returns false if the key definitely does not exist in this SSTable.
func (r *Reader) MayContain(key []byte) bool {
	return r.bloom.Contains(key)
}

// Get finds a key in the SSTable. Returns (value, isTombstone, found, error).
func (r *Reader) Get(key []byte) ([]byte, bool, bool, error) {
	if !r.MayContain(key) || len(r.index) == 0 {
		return nil, false, false, nil // Bloom fast negative path: 0 I/O
	}

	if bytes.Compare(key, r.index[0].Key) < 0 {
		return nil, false, false, nil
	}

	// Binary search sparse index to find candidate block
	idx := sort.Search(len(r.index), func(i int) bool {
		return bytes.Compare(r.index[i].Key, key) > 0
	})
	if idx > 0 {
		idx--
	}
	if idx >= len(r.index) {
		return nil, false, false, nil
	}

	handle := r.index[idx].Handle
	rawBlock := make([]byte, handle.Length)
	if _, err := r.file.ReadAt(rawBlock, int64(handle.Offset)); err != nil {
		return nil, false, false, err
	}

	if len(rawBlock) < 4 {
		return nil, false, false, errors.New("sstable data block too short")
	}

	data := rawBlock[:len(rawBlock)-4]
	expectedCRC := binary.BigEndian.Uint32(rawBlock[len(rawBlock)-4:])
	if crc32.Checksum(data, crcTable) != expectedCRC {
		return nil, false, false, errors.New("sstable data block crc mismatch")
	}

	// Scan entries within the block
	buf := bytes.NewReader(data)
	hdr := make([]byte, 7)
	for buf.Len() > 0 {
		if _, err := buf.Read(hdr); err != nil {
			break
		}
		kLen := int(binary.BigEndian.Uint16(hdr[0:2]))
		vLen := int(binary.BigEndian.Uint32(hdr[2:6]))
		isTomb := hdr[6] == 1

		k := make([]byte, kLen)
		if _, err := buf.Read(k); err != nil {
			break
		}
		v := make([]byte, vLen)
		if vLen > 0 {
			if _, err := buf.Read(v); err != nil {
				break
			}
		}

		if bytes.Equal(k, key) {
			return v, isTomb, true, nil
		}
	}

	return nil, false, false, nil
}

// Close closes the file descriptor.
func (r *Reader) Close() error {
	if r.file != nil {
		err := r.file.Close()
		r.file = nil
		return err
	}
	return nil
}

// FileSize returns the size of the SSTable file on disk.
func (r *Reader) FileSize() int64 {
	return r.fileSize
}

// EntryCount returns total records in SSTable.
func (r *Reader) EntryCount() uint64 {
	return r.footer.EntryCount
}
