package bgen

import (
	"io"
)

// Via https://play.golang.org/p/rn0bAjeEGtK

type bitReader struct {
	reader io.ByteReader
	byte   byte
	offset byte

	errCache    error
	lastBit     bool
	resultCache uint64
}

func newBitReader(r io.ByteReader) *bitReader {
	return &bitReader{r, 0, 0, nil, false, 0}
}

func (r *bitReader) ReadBit() (bool, error) {
	if r.offset == 8 {
		r.offset = 0
	}
	if r.offset == 0 {
		if r.byte, r.errCache = r.reader.ReadByte(); r.errCache != nil {
			return false, r.errCache
		}
	}
	r.lastBit = (r.byte & (0x80 >> r.offset)) != 0
	r.offset++
	return r.lastBit, nil
}

func (r *bitReader) ReadUint(nbits int) (uint64, error) {
	r.resultCache = 0
	for i := nbits - 1; i >= 0; i-- {
		r.lastBit, r.errCache = r.ReadBit()
		if r.errCache != nil {
			return 0, r.errCache
		}
		if r.lastBit {
			r.resultCache |= 1 << uint(i)
		}
	}
	return r.resultCache, nil
}

func (r *bitReader) ReadUintLittleEndian(nbits int) (uint64, error) {
	// Bit order is good
	// Byte order is bad
	// Collect bytes
	// Reverse their order

	loops := nbits / 8
	remainder := nbits % 8

	r.resultCache = 0

	for loop := 0; loop < loops; loop++ {
		for i := 8 - 1; i >= 0; i-- {
			r.lastBit, r.errCache = r.ReadBit()
			if r.errCache != nil {
				return 0, r.errCache
			}
			if r.lastBit {
				r.resultCache |= 1 << uint(i+(8*loop))
			}
		}
	}
	if remainder > 0 {
		for i := remainder - 1; i >= 0; i-- {
			r.lastBit, r.errCache = r.ReadBit()
			if r.errCache != nil {
				return 0, r.errCache
			}
			if r.lastBit {
				r.resultCache |= 1 << uint(i+(8*loops))
			}
		}
	}

	return r.resultCache, nil
}
