package bgen

// This variant is inspired by the C version from
// https://git.biohpc.swmed.edu/zhanxw/rvtests For a more direct translation,
// see this Golang playground example: https://play.golang.org/p/l4uNS0G5KzU
type bitReader struct {
	offset int
	bytes  []byte
	nybble int
}

func newBitReader(bytes []byte, nybbleSize int) *bitReader {
	br := &bitReader{
		bytes:  bytes,
		nybble: nybbleSize,
	}

	return br
}

func (br *bitReader) Next() uint32 {
	if br.nybble == 8 {
		// Speed up the most common path
		br.offset += 8
		return uint32(br.bytes[(br.offset-8)/8])
	}
	var result uint32
	for i := 0; i < br.nybble; i++ {
		result |= br.getBit(i) << uint32(i)
	}
	br.offset += br.nybble

	return result
}

func (br *bitReader) getBit(idx int) uint32 {
	// whichByte := (br.offset + idx) / 8
	// remaining := (br.offset + idx) % 8
	if br.bytes[(br.offset+idx)/8]&(1<<uint((br.offset+idx)%8)) != 0 {
		return 1
	}

	return 0
}
