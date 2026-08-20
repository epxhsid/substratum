// SPDX-License-Identifier: MIT
// package bloom provides a Bloom filter implementation.
package bloom

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/spaolacci/murmur3"
)

type BloomFilter struct {
	bitset []byte
	k      uint32
}

// NewBloomFilter creates a Bloom filter sized for approximately ek (estimated key) elements
// with a target false-positive rate of fpr (false positive rate)
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
// Bloom filter. h2 uses a fixed Murmur3 seed.
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

// serializes the BloomFilter into a byte slice/binary representation.
// the serialized format is: [magic][version][k][bitset]
// all integer fields are serialized in big-endian order.
// the bitset contains the filter's raw bit vector.
func (bf *BloomFilter) Marshal() ([]byte, error) {
	if bf == nil {
		return nil, errors.New("lsm: nil bloom filter")
	}

	if len(bf.bitset) == 0 {
		return nil, errors.New("lsm: empty bloom filter")
	}

	if bf.k == 0 {
		return nil, errors.New("lsm: invalid bloom filter hash count")
	}

	buf := make([]byte, 12+len(bf.bitset))

	binary.BigEndian.PutUint32(buf[0:4], bloomMagic)
	binary.BigEndian.PutUint32(buf[4:8], bloomVersion)
	binary.BigEndian.PutUint32(buf[8:12], bf.k)

	copy(buf[12:], bf.bitset)

	return buf, nil
}

// deserializes the BloomFilter from a byte slice/binary representation.
func (bf *BloomFilter) Unmarshal(data []byte) error {
	if bf == nil {
		return errors.New("lsm: nil bloom filter")
	}

	if len(data) < 12 {
		return errors.New("lsm: invalid bloom filter data")
	}

	magic := binary.BigEndian.Uint32(data[0:4])
	if magic != bloomMagic {
		return errors.New("lsm: invalid bloom filter magic")
	}

	version := binary.BigEndian.Uint32(data[4:8])
	if version != bloomVersion {
		return fmt.Errorf(
			"lsm: unsupported bloom filter version: %d",
			version,
		)
	}

	k := binary.BigEndian.Uint32(data[8:12])
	if k == 0 {
		return errors.New("lsm: invalid bloom filter hash count")
	}

	bitset := data[12:]
	if len(bitset) == 0 {
		return errors.New("lsm: empty bloom filter")
	}

	bf.k = k
	bf.bitset = make([]byte, len(bitset))
	copy(bf.bitset, bitset)

	return nil
}
