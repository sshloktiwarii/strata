package bloom

import (
	"fmt"
	"testing"
)

func TestBloomFilterBasic(t *testing.T) {
	bf := New(1000, 0.01)

	// Add 500 keys
	for i := 0; i < 500; i++ {
		key := []byte(fmt.Sprintf("key-%06d", i))
		bf.Add(key)
	}

	// All added keys must be present (Zero false negatives)
	for i := 0; i < 500; i++ {
		key := []byte(fmt.Sprintf("key-%06d", i))
		if !bf.Contains(key) {
			t.Fatalf("expected key %s to be present in bloom filter", key)
		}
	}

	// Non-present keys should have very low false positive rate (< 3%)
	falsePositives := 0
	trials := 5000
	for i := 500; i < 500+trials; i++ {
		key := []byte(fmt.Sprintf("key-%06d", i))
		if bf.Contains(key) {
			falsePositives++
		}
	}

	fpRate := float64(falsePositives) / float64(trials)
	t.Logf("Measured FP Rate: %.4f (target: 0.01)", fpRate)
	if fpRate > 0.03 {
		t.Fatalf("false positive rate too high: got %.4f, want <= 0.03", fpRate)
	}
}

func TestBloomFilterSerialization(t *testing.T) {
	bf := New(200, 0.01)
	for i := 0; i < 150; i++ {
		bf.Add([]byte(fmt.Sprintf("item-%d", i)))
	}

	encoded := bf.Encode()
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("failed to decode bloom filter: %v", err)
	}

	for i := 0; i < 150; i++ {
		key := []byte(fmt.Sprintf("item-%d", i))
		if !decoded.Contains(key) {
			t.Fatalf("decoded bloom filter missing key %s", key)
		}
	}

	absentKey := []byte("non-existent-arbitrary-key")
	if decoded.Contains(absentKey) {
		t.Logf("Warning: absent key hit false positive (allowed probabilistically)")
	}
}
