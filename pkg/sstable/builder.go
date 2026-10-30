package sstable

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"

	"github.com/sshloktiwarii/lsm-tree-db/pkg/bloom"
)

var crcTable = crc32.MakeTable(crc32.Castagnoli)

// Builder builds an immutable SSTable file on disk.
type Builder struct {
	file        *os.File
	path        string
	blockSize   int
	currBlock   *bytes.Buffer
	firstKey    []byte
	index       []IndexEntry
	bloom       *bloom.BloomFilter
	offset      uint64
	entryCount  uint64
	minKey      []byte
	maxKey      []byte
}

// NewBuilder initializes a new SSTable builder.
func NewBuilder(path string, expectedEntries int, blockSize int) (*Builder, error) {
	if blockSize <= 0 {
		blockSize = DefaultBlockSize
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return &Builder{
		file:      f,
		path:      path,
		blockSize: blockSize,
		currBlock: bytes.NewBuffer(make([]byte, 0, blockSize)),
		bloom:     bloom.New(expectedEntries, 0.01),
	}, nil
}

// Add appends a key-value entry (keys must be in sorted ascending order).
func (b *Builder) Add(key, value []byte, isTombstone bool) error {
	if len(b.firstKey) == 0 {
		b.firstKey = bytes.Clone(key)
	}
	if len(b.minKey) == 0 {
		b.minKey = bytes.Clone(key)
	}
	b.maxKey = bytes.Clone(key)
	b.bloom.Add(key)
	b.entryCount++

	tombByte := byte(0)
	if isTombstone {
		tombByte = 1
	}

	entryHdr := make([]byte, 2+4+1)
	binary.BigEndian.PutUint16(entryHdr[0:2], uint16(len(key)))
	binary.BigEndian.PutUint32(entryHdr[2:6], uint32(len(value)))
	entryHdr[6] = tombByte

	b.currBlock.Write(entryHdr)
	b.currBlock.Write(key)
	b.currBlock.Write(value)

	if b.currBlock.Len() >= b.blockSize {
		return b.flushBlock()
	}
	return nil
}

func (b *Builder) flushBlock() error {
	if b.currBlock.Len() == 0 {
		return nil
	}
	data := b.currBlock.Bytes()
	checksum := crc32.Checksum(data, crcTable)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, checksum)

	blockLen := uint64(len(data) + 4)
	if _, err := b.file.Write(data); err != nil {
		return err
	}
	if _, err := b.file.Write(crcBuf); err != nil {
		return err
	}

	b.index = append(b.index, IndexEntry{
		Key: b.firstKey,
		Handle: BlockHandle{
			Offset: b.offset,
			Length: blockLen,
		},
	})

	b.offset += blockLen
	b.currBlock.Reset()
	b.firstKey = nil
	return nil
}

// Finish flushes remaining data, writes bloom filter, index block, and footer.
func (b *Builder) Finish() error {
	if err := b.flushBlock(); err != nil {
		b.file.Close()
		return err
	}

	// 1. Write Bloom Filter
	bloomBytes := b.bloom.Encode()
	bloomOffset := b.offset
	bloomLen := uint64(len(bloomBytes))
	if _, err := b.file.Write(bloomBytes); err != nil {
		b.file.Close()
		return err
	}
	b.offset += bloomLen

	// 2. Write Index Section
	indexBuf := new(bytes.Buffer)
	for _, idx := range b.index {
		kLenBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(kLenBuf, uint16(len(idx.Key)))
		indexBuf.Write(kLenBuf)
		indexBuf.Write(idx.Key)

		handleBuf := make([]byte, 16)
		binary.BigEndian.PutUint64(handleBuf[0:8], idx.Handle.Offset)
		binary.BigEndian.PutUint64(handleBuf[8:16], idx.Handle.Length)
		indexBuf.Write(handleBuf)
	}

	idxBytes := indexBuf.Bytes()
	indexOffset := b.offset
	indexLen := uint64(len(idxBytes))
	if _, err := b.file.Write(idxBytes); err != nil {
		b.file.Close()
		return err
	}
	b.offset += indexLen

	// 3. Write Footer (48 bytes)
	footerBuf := make([]byte, FooterSize)
	binary.BigEndian.PutUint64(footerBuf[0:8], indexOffset)
	binary.BigEndian.PutUint64(footerBuf[8:16], indexLen)
	binary.BigEndian.PutUint64(footerBuf[16:24], bloomOffset)
	binary.BigEndian.PutUint64(footerBuf[24:32], bloomLen)
	binary.BigEndian.PutUint64(footerBuf[32:40], b.entryCount)
	binary.BigEndian.PutUint64(footerBuf[40:48], MagicNumber)

	if _, err := b.file.Write(footerBuf); err != nil {
		b.file.Close()
		return err
	}

	if err := b.file.Sync(); err != nil {
		b.file.Close()
		return err
	}
	return b.file.Close()
}
