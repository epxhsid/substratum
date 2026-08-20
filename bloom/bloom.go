// SPDX-License-Identifier: MIT
// package bloom provides a Bloom filter implementation.

package bloom

import (
	"fmt"
	"math"
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
