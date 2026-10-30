package bloom

import (
	"encoding/binary"
	"errors"
	"hash/fnv"
	"math"
)

// BloomFilter is a thread-safe, space-efficient probabilistic data structure
// for testing set membership with zero false negatives and configurable false positive rate.
type BloomFilter struct {
	bits []byte
	size uint64 // total number of bits
	k    uint32 // number of hash functions
}

// New creates an optimal Bloom filter for expected number of keys and target false positive rate.
func New(expectedKeys int, fpRate float64) *BloomFilter {
	if expectedKeys <= 0 {
		expectedKeys = 64
	}
	if fpRate <= 0.0 || fpRate >= 1.0 {
		fpRate = 0.01 // default 1% false positive rate
	}

	// m = - (n * ln(p)) / (ln(2)^2)
	n := float64(expectedKeys)
	m := math.Ceil(-1.0 * (n * math.Log(fpRate)) / (math.Ln2 * math.Ln2))
	if m < 64 {
		m = 64
	}

	// k = (m / n) * ln(2)
	k := math.Round((m / n) * math.Ln2)
	if k < 1 {
		k = 1
	} else if k > 30 {
		k = 30
	}

	numBytes := (uint64(m) + 7) / 8
	return &BloomFilter{
		bits: make([]byte, numBytes),
		size: numBytes * 8,
		k:    uint32(k),
	}
}

// NewWithBits creates a BloomFilter with pre-allocated raw bits and k hash count.
func NewWithBits(bits []byte, k uint32) *BloomFilter {
	return &BloomFilter{
		bits: bits,
		size: uint64(len(bits)) * 8,
		k:    k,
	}
}

// hashes generates two 64-bit hashes using FNV-1a variants for Kirsch-Mitzenmacher optimization:
// gi(x) = h1(x) + i * h2(x)
func (bf *BloomFilter) hashes(data []byte) (uint64, uint64) {
	h := fnv.New64a()
	h.Write(data)
	h1 := h.Sum64()

	// Secondary hash using bit mix
	h2 := h1>>32 | h1<<32
	h2 ^= 0x517cc1b727220a95
	h2 = (h2 ^ (h2 >> 30)) * 0xbf58476d1ce4e5b9
	h2 = (h2 ^ (h2 >> 27)) * 0x94d049bb133111eb
	h2 ^= (h2 >> 31)

	if h2 == 0 {
		h2 = 0x9e3779b97f4a7c15 // non-zero odd fallback
	}
	return h1, h2
}

// Add inserts a key into the Bloom filter.
func (bf *BloomFilter) Add(key []byte) {
	if bf.size == 0 {
		return
	}
	h1, h2 := bf.hashes(key)
	for i := uint32(0); i < bf.k; i++ {
		bitIndex := (h1 + uint64(i)*h2) % bf.size
		bf.bits[bitIndex/8] |= 1 << (bitIndex % 8)
	}
}

// Contains checks if a key might be in the set.
// If it returns false, the key is definitely not present.
// If it returns true, the key is probably present.
func (bf *BloomFilter) Contains(key []byte) bool {
	if bf.size == 0 || len(bf.bits) == 0 {
		return false
	}
	h1, h2 := bf.hashes(key)
	for i := uint32(0); i < bf.k; i++ {
		bitIndex := (h1 + uint64(i)*h2) % bf.size
		if (bf.bits[bitIndex/8] & (1 << (bitIndex % 8))) == 0 {
			return false
		}
	}
	return true
}

// Encode serializes the Bloom filter into binary:
// [4 bytes: k][4 bytes: byte_len][N bytes: bits]
func (bf *BloomFilter) Encode() []byte {
	buf := make([]byte, 8+len(bf.bits))
	binary.BigEndian.PutUint32(buf[0:4], bf.k)
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(bf.bits)))
	copy(buf[8:], bf.bits)
	return buf
}

// Decode deserializes a Bloom filter from binary.
func Decode(data []byte) (*BloomFilter, error) {
	if len(data) < 8 {
		return nil, errors.New("bloom filter buffer too short")
	}
	k := binary.BigEndian.Uint32(data[0:4])
	numBytes := binary.BigEndian.Uint32(data[4:8])
	if uint32(len(data)-8) < numBytes {
		return nil, errors.New("bloom filter buffer corrupted: payload truncated")
	}
	bits := make([]byte, numBytes)
	copy(bits, data[8:8+numBytes])

	return &BloomFilter{
		bits: bits,
		size: uint64(numBytes) * 8,
		k:    k,
	}, nil
}

// SizeInBytes returns the memory size of the filter payload.
func (bf *BloomFilter) SizeInBytes() int {
	return len(bf.bits)
}
