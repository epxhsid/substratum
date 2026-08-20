// SPDX-License-Identifier: MIT
// package bloom provides a Bloom filter implementation.
package bloom

import (
	"fmt"
	"math"

	"github.com/spaolacci/murmur3"
)

type BloomFilter struct {
	bitset []byte
	k      uint32
}

func NewBloomFilter(ek int, fpr float64) (*BloomFilter, error) {
	if ek <= 0 {
		return nil, fmt.Errorf("not positive")
	}

	if fpr <= 0 || fpr >= 1 {
		return nil, fmt.Errorf("false positive rate must be between 0 and 1")
	}

	// function for calculating the bit array size (m): "m = -(n * ln(p)) / (ln(2)^2)"
	// Calculates the optimal number of bits (m) for a Bloom filter given the
	// expected number of keys (n) and the target false positive rate (p).
	//
	// when decreasing the fpr (e.g. from 0.1 to 0.01), math.Log(fpr) becomes
	// a larger negative natural log, which correctly scales up to a larger m,
	// to allocate more memory thus reducing the false positive rate
	m := int(math.Ceil(-(float64(ek) * math.Log(fpr)) / ln2Sq))

	// function for calculating the number of hash functions (k): "k = (m / n) * ln(2)"
	// Calculates the optimal number of hash functions (k) for a Bloom filter given the
	// bit array size (m) and the expected number of keys (n).
	//
	// balancing the ratio of bits to elements with the natural logarithm of 2
	// finds the optimal number of hash functions to minimize the false positive
	// rate. minimum value of 1 is placed for k, to avoid division by zero and
	// ensure at least one hash function is always used for hashing/filtering
	k := max(1, uint32(math.Round((float64(m)/float64(ek))*math.Log(2))))

	return &BloomFilter{
		bitset: make([]byte, (m+7)/8),
		k:      k,
	}, nil
}

// Kirsch-Mitzenmacher optimization: derive k bit positions from two
// Murmur3 hashes rather than computing k separate hashes.
//
// formula: g_i(x) = (h1(x) + i*h2(x)) mod m
//
// m is the number of bits in the filter and i ∈ [0, k).
// h1 and h2 are independent hash functions for the purposes of the
// Bloom filter. h2 uses a fixed Murmur3 seed (2538058380).
func (bf *BloomFilter) Add(key string) {
	if bf == nil || len(bf.bitset) == 0 {
		return
	}

	h1 := murmur3.Sum64([]byte(key))
	h2 := murmur3.Sum64WithSeed([]byte(key), 0x9747b28c)

	bc := uint64(len(bf.bitset) * 8)

	for i := uint32(0); i < bf.k; i++ {
		hash := h1 + uint64(i)*h2
		bit := hash % bc

		bf.bitset[bit/8] |= 1 << (bit % 8)
	}
}

// Contains returns true if the key is likely to be in the filter, false otherwise.
func (bf *BloomFilter) Contains(key string) bool {
	if bf == nil || len(bf.bitset) == 0 {
		return false
	}

	h1 := murmur3.Sum64([]byte(key))
	h2 := murmur3.Sum64WithSeed([]byte(key), 0x9747b28c)

	bc := uint64(len(bf.bitset) * 8)

	for i := uint32(0); i < bf.k; i++ {
		hash := h1 + uint64(i)*h2
		bit := hash % bc

		if bf.bitset[bit/8]&(1<<(bit%8)) == 0 {
			return false
		}
	}

	return true
}
