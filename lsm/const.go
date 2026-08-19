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

	bloomVersion    = 0x01
	bloomHeaderSize = 17
)

const ln2Sq = 0.4804530139182014 // math.Log(2) * math.Log(2)
