package utilities

func BytesToShort(b []uint8) uint16 {
	return (uint16(b[1]) << 8) | uint16(b[0])
}

func ShortToBytes(s uint16) []uint8 {
	return []uint8{uint8(s & 0xFF), uint8((s & 0xFF00) >> 8)}
}

func BytesToInt(b []uint8) uint32 {
	return (uint32(b[3]) << 24) | (uint32(b[2]) << 16) |
		(uint32(b[1]) << 8) | (uint32(b[0]))
}

func IntToBytes(s uint32) []uint8 {
	return []uint8{
		uint8(s & 0xFF),
		uint8((s & 0xFF00) >> 8),
		uint8((s & 0xFF0000) >> 16),
		uint8((s & 0xFF000000) >> 24),
	}
}

func DirClusterToUint(cluster_lo uint, cluster_hi uint) uint32 {
	return uint32((cluster_hi << 16) | cluster_lo)
}

func YearToFATYear(year int) int {
	return year - 1980
}
