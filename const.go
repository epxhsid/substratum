// SPDX-License-Identifier: MIT
package lsm

const (
	opSet    byte = 0x01
	opDelete byte = 0x02
)

const (
	sparseIndexInterval = 16
	walFlushThreshold   = 4096

	maxKeySize    = 64 << 10
	maxValueSize  = 64 << 20
	maxRecordSize = maxKeySize + maxValueSize + 9
)

const (
	bloomMagic   uint32 = 0x424C4D46 // "B L M F"
	bloomVersion uint32 = 1
)

// formula: math.Log(2) * math.Log(2)
// Define the pre-calculated constant for (ln(2))^2 to optimize performance.
const ln2Sq = 0.4804530139182014
