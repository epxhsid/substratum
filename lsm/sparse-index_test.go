package lsm

import "testing"

func TestSparseIndexOffset(t *testing.T) {
	index := SparseIndex{
		entries: []sparseIndexEntry{
			{
				key:    "alpha",
				offset: 0,
			},
			{
				key:    "delta",
				offset: 64,
			},
			{
				key:    "kappa",
				offset: 128,
			},
		},
	}

	tests := []struct {
		key    string
		offset int64
	}{
		{
			key:    "aardvark",
			offset: 0,
		},
		{
			key:    "alpha",
			offset: 0,
		},
		{
			key:    "beta",
			offset: 0,
		},
		{
			key:    "delta",
			offset: 64,
		},
		{
			key:    "lambda",
			offset: 128,
		},
	}

	for _, tc := range tests {
		if got := index.offset(tc.key); got != tc.offset {
			t.Fatalf("offset(%q) = %d, want %d", tc.key, got, tc.offset)
		}
	}
}
