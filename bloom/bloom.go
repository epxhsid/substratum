package bloom

type BloomFilter struct {
	bitset []byte
	k      uint32
}

func NewBloomFilter(k uint32) *BloomFilter {
	return &BloomFilter{
		bitset: make([]byte, 0),
		k:      k,
	}
}
