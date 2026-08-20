// SPDX-License-Identifier: MIT
// package bloom provides a Bloom filter implementation.
package bloom

const (
	bloomMagic   uint32 = 0x424C4D46 // "B L M F"
	bloomVersion uint32 = 1
)

// formula: math.Log(2) * math.Log(2)
// Define the pre-calculated constant for (ln(2))^2 to optimize performance.
const ln2Sq = 0.4804530139182014
