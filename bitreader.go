package bgen

import (
	"io"
)

// Via https://play.golang.org/p/rn0bAjeEGtK

type bitReader struct {
	reader io.ByteReader
	byte   byte
	offset byte
}

func newBitReader(r io.ByteReader) *bitReader {
	return &bitReader{r, 0, 0}
}

func (r *bitReader) ReadBit() (bool, error) {
	if r.offset == 8 {
		r.offset = 0
	}
	if r.offset == 0 {
		var err error
		if r.byte, err = r.reader.ReadByte(); err != nil {
			return false, err
		}
	}
	bit := (r.byte & (0x80 >> r.offset)) != 0
	r.offset++
	return bit, nil
}

func (r *bitReader) ReadUint(nbits int) (uint64, error) {
	var result uint64
	var bit bool
	var err error
	for i := nbits - 1; i >= 0; i-- {
		bit, err = r.ReadBit()
		if err != nil {
			return 0, err
		}
		if bit {
			result |= 1 << uint(i)
		}
	}
	return result, nil
}

func (r *bitReader) ReadUintLittleEndian(nbits int) (final uint64, err error) {
	// Bit order is good
	// Byte order is bad
	// Collect bytes
	// Reverse their order

	loops := nbits / 8
	remainder := nbits % 8

	var bit bool
	for loop := 0; loop < loops; loop++ {
		for i := 8 - 1; i >= 0; i-- {
			bit, err = r.ReadBit()
			if err != nil {
				return 0, err
			}
			if bit {
				final |= 1 << uint(i+(8*loop))
			}
		}
	}
	if remainder > 0 {
		for i := remainder - 1; i >= 0; i-- {
			bit, err := r.ReadBit()
			if err != nil {
				return 0, err
			}
			if bit {
				final |= 1 << uint(i+(8*loops))
			}
		}
	}

	return
}
